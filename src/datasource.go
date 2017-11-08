package main

import(
	"log"
	
	databox "github.com/cgreenhalgh/lib-go-databox"
)

// datasource utility stuff

const (
	API_TIMESERIES = iota
	API_KEYVALUE
)
type ApiType int

// about one datasource
type DatasourceInfo struct{
	Metadata databox.StoreMetadata
	Api ApiType
	DataStoreHref string
	TsApi databox.TimeSeries_0_2_0
	KvApi databox.KeyValue_0_2_0
}

func (d *Driver) FindDatasource(id string) *DatasourceInfo {
	for i:=0; i<len(d.datasources); i++ {
		ds := &d.datasources[i]
		if ds.Metadata.DataSourceID == id {
			return ds
		}
	}
	log.Printf("ERROR: did not find datasource %s", id)
	return nil
}

func (d *Driver) registerDatasources() {

	for i:=0; i<len(d.datasources); i++ {
		ds := &d.datasources[i]
		_,err := databox.RegisterDatasource(ds.DataStoreHref, ds.Metadata)
		if err != nil {
			d.LogFatalError("Error registering "+ds.Metadata.DataSourceID+" datasource", err)
		} else {
			log.Printf("registered datasource %s", ds.Metadata.DataSourceID)
		}
	}
}

func (d *Driver) makeDatasourceApis() {

	for i:=0; i<len(d.datasources); i++ {
		ds := &d.datasources[i]
		var err error
		switch ds.Api {
		case API_TIMESERIES:
			ds.TsApi,err = databox.MakeStoreTimeSeries_0_2_0(ds.DataStoreHref, ds.Metadata.DataSourceID, ds.Metadata.StoreType)
		case API_KEYVALUE:
			ds.KvApi,err = databox.MakeStoreKeyValue_0_2_0(ds.DataStoreHref, ds.Metadata.DataSourceID, ds.Metadata.StoreType)
		}
		if err != nil {
			log.Printf("Error creating api for datasource %s, type %d, store %s: %s", ds.Metadata.DataSourceID, ds.Api, ds.Metadata.StoreType, err.Error())
		}
	}
}