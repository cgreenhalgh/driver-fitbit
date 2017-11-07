package main

import(
	"log"
	
	databox "github.com/cgreenhalgh/lib-go-databox"
)

// datasource utility stuff

// about one datasource
type DatasourceInfo struct{
	Metadata databox.StoreMetadata
	Timeseries bool
	DataStoreHref string
}

func (d *Driver) registerDatasources() {

	for i:=0; i<len(d.datasources); i++ {
		ds := d.datasources[i]
		_,err := databox.RegisterDatasource(ds.DataStoreHref, ds.Metadata)
		if err != nil {
			d.LogFatalError("Error registering "+ds.Metadata.DataSourceID+" datasource", err)
		} else {
			log.Printf("registered datasource %s", ds.Metadata.DataSourceID)
		}
	}
}