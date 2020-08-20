module github.com/meshplus/pier-client-fabric

go 1.13

require (
	github.com/Rican7/retry v0.1.0
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/cloudflare/cfssl v0.0.0-20190409034051-768cd563887f
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/golang/protobuf v1.4.0
	github.com/google/certificate-transparency-go v1.1.0 // indirect
	github.com/hashicorp/go-plugin v1.3.0
	github.com/hyperledger/fabric v2.0.1+incompatible
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a
	github.com/hyperledger/fabric-lib-go v1.0.0 // indirect
	github.com/hyperledger/fabric-protos-go v0.0.0-20200330074707-cfe579e86986
	github.com/hyperledger/fabric-sdk-go v1.0.0-alpha5
	github.com/meshplus/bitxhub-kit v1.0.1-0.20200813124031-6f6bdc99564f
	github.com/meshplus/bitxhub-model v1.0.0-rc4.0.20200731025300-2bb1717059e0
	github.com/meshplus/pier v1.1.0-rc1.0.20200824115625-bb57600455be
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/viper v1.6.1
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.2
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.3
)
