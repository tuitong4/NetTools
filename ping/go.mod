module ping

go 1.13

require (
	github.com/go-ini/ini v1.55.0
	github.com/hprose/hprose-golang v2.0.5+incompatible
)

require (
	github.com/smartystreets/goconvey v1.6.4 // indirect
	gopkg.in/ini.v1 v1.55.0 // indirect
	local.lc/log v0.0.0
)

replace local.lc/log => ./vendor/local.lc/log

replace github.com/Shopify/sarama => ./vendor/github.com/sarama/sarama
