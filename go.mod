module github.com/meshplus/pier-client-fabric

go 1.13

require (
	github.com/OneOfOne/xxhash v1.2.5 // indirect
	github.com/Rican7/retry v0.1.0
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/cloudflare/cfssl v0.0.0-20190409034051-768cd563887f
	github.com/elastic/gosigar v0.8.1-0.20180330100440-37f05ff46ffa // indirect
	github.com/ethereum/go-ethereum v1.9.18 // indirect
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/golang/protobuf v1.4.0
	github.com/google/certificate-transparency-go v1.1.0 // indirect
	github.com/hashicorp/go-plugin v1.3.0
	github.com/hyperledger/fabric v2.0.1+incompatible
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a
	github.com/hyperledger/fabric-lib-go v1.0.0 // indirect
	github.com/hyperledger/fabric-protos-go v0.0.0-20200330074707-cfe579e86986
	github.com/hyperledger/fabric-sdk-go v1.0.0-alpha5
	github.com/meshplus/bitxhub v1.0.0-rc2 // indirect
	github.com/meshplus/bitxhub-kit v1.1.2-0.20201203072410-8a0383a6870d
	github.com/meshplus/bitxhub-model v1.1.2-0.20210312014622-c3ad532b64ad
	github.com/meshplus/pier v1.5.1-0.20210312103925-148435c71325
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/viper v1.6.1
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.2
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.3
)
