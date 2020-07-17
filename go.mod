module github.com/meshplus/pier-client-fabric

go 1.13

require (
	github.com/Rican7/retry v0.1.0
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/cloudflare/cfssl v0.0.0-20180223231731-4e2dcbde5004
	github.com/golang/protobuf v1.4.0
	github.com/google/certificate-transparency-go v1.1.0 // indirect
	github.com/hashicorp/go-plugin v1.3.0
	github.com/hyperledger/fabric v2.0.1+incompatible
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a
	github.com/hyperledger/fabric-lib-go v1.0.0 // indirect
	github.com/hyperledger/fabric-protos-go v0.0.0-20200330074707-cfe579e86986
	github.com/hyperledger/fabric-sdk-go v1.0.0-alpha5
	github.com/meshplus/bitxhub-kit v1.0.1-0.20200525112026-df2160653e23
	github.com/meshplus/bitxhub-model v1.0.0-rc4.0.20200707045101-18b88b80efb1
	github.com/meshplus/pier v1.0.0-rc1.0.20200717044435-de24cfbef0f3
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/viper v1.6.1
)

replace github.com/golang/protobuf => github.com/golang/protobuf v1.3.2
