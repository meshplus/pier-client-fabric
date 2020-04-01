module github.com/meshplus/pier-client-fabric

go 1.13

require (
	github.com/Knetic/govaluate v3.0.0+incompatible // indirect
	github.com/Rican7/retry v0.1.0
	github.com/Shopify/sarama v1.26.1 // indirect
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/cloudflare/cfssl v0.0.0-20180223231731-4e2dcbde5004
	github.com/fsouza/go-dockerclient v1.6.3 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/google/certificate-transparency-go v1.1.0 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/hyperledger/fabric v1.4.6
	github.com/hyperledger/fabric-amcl v0.0.0-20200128223036-d1aa2665426a // indirect
	github.com/hyperledger/fabric-lib-go v1.0.0 // indirect
	github.com/hyperledger/fabric-sdk-go v1.0.0-alpha5
	github.com/meshplus/bitxhub-kit v1.0.0-rc1
	github.com/meshplus/bitxhub-model v1.0.0-rc1
	github.com/meshplus/pier v0.0.0-00010101000000-000000000000
	github.com/miekg/pkcs11 v1.0.3 // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/viper v1.4.0
	github.com/sykesm/zap-logfmt v0.0.3 // indirect
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.7

replace golang.org/x/net => golang.org/x/net v0.0.0-20200202094626-16171245cfb2

replace github.com/meshplus/pier => ../pier

replace golang.org/x/text => golang.org/x/text v0.3.2

replace github.com/spf13/afero => github.com/spf13/afero v1.1.2

replace github.com/pelletier/go-toml => github.com/pelletier/go-toml v1.2.0

replace github.com/spf13/jwalterweatherman => github.com/spf13/jwalterweatherman v1.0.0

replace github.com/mholt/archiver => github.com/mholt/archiver v0.0.0-20180417220235-e4ef56d48eb0
