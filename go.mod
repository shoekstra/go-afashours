module github.com/shoekstra/go-afashours

go 1.15

replace github.com/dougEfresh/gtoggl-api => github.com/shoekstra/gtoggl-api v0.0.0-20201223232228-04d5e2238395

replace github.com/tim-online/go-afas-profit-rest => github.com/shoekstra/go-afas-profit-rest v0.0.0-20201219163557-76c868b71cc3

require (
	github.com/dougEfresh/gtoggl-api v0.0.0-20200303083000-7924af45c7f9
	github.com/gocarina/gocsv v0.0.0-20211203214250-4735fba0c1d9
	github.com/gomodule/redigo v1.8.4 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/throttled/throttled v2.2.5+incompatible // indirect
	github.com/tim-online/go-afas-profit-rest v0.0.0-20200213113110-c6fd6719ce90
)
