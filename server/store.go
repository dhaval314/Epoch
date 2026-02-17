package main

import (
	"log"
	"sync"
	"encoding/json"

	badger "github.com/dgraph-io/badger/v4"
	pb "github.com/dhaval314/epoch/proto"
)

type JobStore struct{
	mu sync.Mutex;
	jobs map[string]JobContext // HashMap to store all the jobs
	db *badger.DB
}

type JobContext struct{
	Status string;
	Output string;
	Job *pb.Job
}

// Initialize the JobStore struct
var store = JobStore{
	jobs : make(map[string]JobContext),
}



func SaveJob(id string, req JobContext, db *badger.DB) error {
   	return db.Update(func(txn *badger.Txn) error {
		jsonData, err := json.Marshal(req)
		if err != nil{
			log.Printf("[-] Error serializing jobContext %v", err)
		}

		return txn.Set([]byte("job:"+id), jsonData)
    })
}

func LoadJobs(db *badger.DB) error{
	// Slice to store zombie jobs, which are later marked as failed
	jobsToFix := []JobContext{}

	err := db.View(func(txn *badger.Txn) error { 

		// Iterate through the database
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		
		// Prefix to ensure we get jobs instead of some other stuff which might be in the db
		prefix := []byte("job:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {

			// Get the current Item
			item := it.Item()
			var jobContext JobContext
			err := item.Value(func(v []byte) error {
				json.Unmarshal(v, &jobContext) // Convert the json back into a struct

				if jobContext.Status == "RUNNING"{
					jobContext.Status = "FAILED" // Since all the running processes wont finish, mark them failed
					jobsToFix = append(jobsToFix, jobContext)
				}
				store.jobs[jobContext.Job.Id] = jobContext // store the jobs in the map
			return nil
			})
		if err != nil {
			return err
		}
		}
		return nil
	})
	if err != nil{
		log.Printf("[-] Error loading jobs %v", err)
	}
	// Update the zombie "RUNNING" processes with "FAILED"
	db.Update(func(txn *badger.Txn) error {
		for _, jobContext := range jobsToFix{
			jsonData, err := json.Marshal(jobContext)
			if err != nil{
				log.Printf("[-] Error serializing jobContext %v", err)
			}
			err = txn.Set([]byte("job:"+jobContext.Job.Id), jsonData)
			if err != nil{
				return err
			}
		}
		return nil
	})
	return nil
}




func CreateDB()(*badger.DB, error){
	db, err := badger.Open(badger.DefaultOptions("./badger"))
  	if err != nil {
    	return nil, err
  	}
	return db, nil
}