module github.com/meshplus/pier-client-fabric

go 1.13

require (
	github.com/OneOfOne/xxhash v1.2.5 // indirect
	github.com/Rican7/retry v0.1.0
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/cloudflare/cfssl v0.0.0-20190409034051-768cd563887f
	github.com/elastic/gosigar v0.8.1-0.20180330100440-37f05ff46ffa // indirect
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/certificate-transparency-go v1.1.0 // indirect
	github.com/hashicorp/go-hclog v0.0.0-20180709165350-ff2cf002a8dd
	github.com/hashicorp/go-plugin v1.3.0
	github.com/hyperledger/fabric v2.0.1+incompatible
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a
	github.com/hyperledger/fabric-lib-go v1.0.0 // indirect
	github.com/hyperledger/fabric-protos-go v0.0.0-20200330074707-cfe579e86986
	github.com/hyperledger/fabric-sdk-go v1.0.0-alpha5
	github.com/meshplus/bitxhub-model v1.2.1-0.20210811024313-728f913a1397
	github.com/meshplus/bitxid v0.0.0-20210412025850-e0eaf0f9063a
	github.com/meshplus/pier v1.11.1-0.20210825032911-535baa179f69
	github.com/spf13/viper v1.7.0
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.2
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.3
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200513103714-09dca8ec2884
	google.golang.org/protobuf => google.golang.org/protobuf v1.21.0
)
