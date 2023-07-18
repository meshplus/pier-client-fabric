module github.com/meshplus/pier-client-fabric

go 1.13

require (
	github.com/Rican7/retry v0.1.0
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/cloudflare/cfssl v0.0.0-20190409034051-768cd563887f
	github.com/coreos/etcd v3.3.18+incompatible // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/golang/protobuf v1.4.0
	github.com/google/certificate-transparency-go v1.1.0 // indirect
	github.com/hashicorp/go-hclog v0.0.0-20180709165350-ff2cf002a8dd
	github.com/hashicorp/go-plugin v1.3.0
	github.com/hyperledger/fabric v2.0.1+incompatible
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a
	github.com/hyperledger/fabric-lib-go v1.0.0 // indirect
	github.com/hyperledger/fabric-protos-go v0.0.0-20200330074707-cfe579e86986
	github.com/hyperledger/fabric-sdk-go v1.0.0-alpha5
	github.com/meshplus/bitxhub-model v1.1.2-0.20210513074749-d31e04a9f41d
	github.com/meshplus/pier v1.7.1-0.20210521020418-8d397e8d7646
	github.com/olekukonko/tablewriter v0.0.2-0.20190409134802-7e037d187b0c // indirect
	github.com/prometheus/client_golang v1.4.1 // indirect
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/rogpeppe/go-internal v1.5.2 // indirect
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.7.0
	go.etcd.io/bbolt v1.3.3 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.2
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.3
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
)
