module github.com/meshplus/pier-client-fabric

go 1.13

require (
	github.com/Rican7/retry v0.1.0
	github.com/cloudflare/cfssl v0.0.0-20190409034051-768cd563887f
	github.com/ethereum/go-ethereum v1.10.4
	github.com/fatih/color v1.9.0
	github.com/gin-gonic/gin v1.7.4
	github.com/gobuffalo/packd v1.0.0
	github.com/gobuffalo/packr v1.30.1
	github.com/gobuffalo/packr/v2 v2.5.1
	github.com/golang/protobuf v1.5.2
	github.com/google/certificate-transparency-go v1.1.0 // indirect
	github.com/hashicorp/go-hclog v0.0.0-20180709165350-ff2cf002a8dd
	github.com/hashicorp/go-plugin v1.3.0
	github.com/hyperledger/fabric v2.1.1+incompatible
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a
	github.com/hyperledger/fabric-protos-go v0.0.0-20201028172056-a3136dde2354
	github.com/hyperledger/fabric-sdk-go v1.0.0-beta1.0.20200106161850-8f3d32c9d1a6
	github.com/meshplus/bitxhub-kit v1.2.1-0.20220412092457-5836414df781
	github.com/meshplus/bitxhub-model v1.2.1-0.20220803022708-9ab7a71abdbf
	github.com/meshplus/pier v1.24.1-0.20220803023357-8533944f0d08
	github.com/spf13/viper v1.7.1
	github.com/urfave/cli v1.22.1
)

replace (
	github.com/binance-chain/tss-lib => github.com/dawn-to-dusk/tss-lib v1.3.3-0.20220330081758-f404e10a1268
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.2
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.3
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200513103714-09dca8ec2884
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
