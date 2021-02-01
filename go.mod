module github.com/meshplus/pier-client-fabric

go 1.13

require (
	github.com/Rican7/retry v0.1.0
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/cloudflare/cfssl v0.0.0-20190409034051-768cd563887f
	github.com/fsouza/go-dockerclient v1.7.0 // indirect
	github.com/golang/protobuf v1.4.0
	github.com/google/certificate-transparency-go v1.1.0 // indirect
	github.com/hashicorp/go-hclog v0.0.0-20180709165350-ff2cf002a8dd
	github.com/hashicorp/go-plugin v1.3.0
	github.com/hyperledger/fabric v2.0.1+incompatible
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a
	github.com/hyperledger/fabric-lib-go v1.0.0 // indirect
	github.com/hyperledger/fabric-protos-go v0.0.0-20200330074707-cfe579e86986
	github.com/hyperledger/fabric-sdk-go v1.0.0-alpha5
	github.com/meshplus/bitxhub-kit v1.1.2-0.20201023073721-052e6b89ea39
	github.com/meshplus/bitxhub-model v1.1.2-0.20201210083234-c041773b8820
	github.com/meshplus/pier v1.1.0-rc1.0.20201210083726-ee6218fa7dcf
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/viper v1.6.1
)

replace (
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.2
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.3
)
