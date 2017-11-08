package main

import(
	"log"
	"time"
)

// Synchronization worker internals

type DataStoreEntry struct{
	//data 
	Timestamp float64 `json:"timestamp"`
}

type SyncHandler func(d *Driver, accessToken string) (bool, error)

func (d *Driver) syncWorkerServer() {
	for {
		//log.Print("Sync waiting")
		req := <- d.syncRequests
		d.settingsLock.Lock()
		accessToken := d.state.AccessToken
		d.settingsLock.Unlock()
		if accessToken == "" {
			log.Print("Sync(internal) with no access token")
			if req != nil {
				req <- false
			}
		} else {
			log.Print("Sync (internal)...")
			d.settingsLock.Lock()
			d.settings.LastSyncStatus = SYNC_ACTIVE
			d.settingsLock.Unlock()
			res, _ := d.syncHandler(d, accessToken)
			d.settingsLock.Lock()
			if res {
				d.settings.LastSyncStatus = SYNC_SUCCESS
				now := time.Now()
				d.settings.LastSync = now
				d.state.LastSync = now
				d.SaveState()
			} else {
				d.settings.LastSyncStatus = SYNC_FAILURE
			}
			d.settingsLock.Unlock()
			// signal done
			if req != nil {
				req <- res
			}			
		}
	}
}

