module libcore

go 1.17

require (
	github.com/Dreamacro/clash v1.7.1
	github.com/golang/protobuf v1.5.2
	github.com/miekg/dns v1.1.43
	github.com/pkg/errors v0.9.1
	github.com/sagernet/gomobile v0.0.0-20210905032500-701a995ff844
	github.com/sagernet/libping v0.1.1
	github.com/sagernet/sagerconnect v0.1.7
	github.com/sirupsen/logrus v1.8.1
	github.com/ulikunitz/xz v0.5.10
	github.com/v2fly/v2ray-core/v4 v4.43.0
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/sys v0.0.0-20211020174200-9d6173849985
	gvisor.dev/gvisor v0.0.0
)

//replace gvisor.dev/gvisor v0.0.0 => ../gvisor
replace gvisor.dev/gvisor v0.0.0 => github.com/sagernet/gvisor v0.0.0-20211022025201-1cae8baac6b3

replace github.com/v2fly/v2ray-core/v4 v4.43.0 => ../../../v2ray-core

replace github.com/Dreamacro/clash v1.7.1 => github.com/sagernet/clash v1.6.5-0.20210913182617-681dd3780179

require (
	github.com/ClashDotNetFramework/go-shadowsocks2 v0.1.8 // indirect
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/dgryski/go-camellia v0.0.0-20191119043421-69a8a13fb23d // indirect
	github.com/dgryski/go-idea v0.0.0-20170306091226-d2fb45a411fb // indirect
	github.com/dgryski/go-metro v0.0.0-20211015221634-2661b20a2446 // indirect
	github.com/dgryski/go-rc2 v0.0.0-20150621095337-8a9021637152 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/geeksbaek/seed v0.0.0-20180909040025-2a7f5fb92e22 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/gofrs/uuid v4.1.0+incompatible // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/jhump/protoreflect v1.10.1 // indirect
	github.com/kierdavis/cfb8 v0.0.0-20180105024805-3a17c36ee2f8 // indirect
	github.com/lucas-clemente/quic-go v0.24.0 // indirect
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40 // indirect
	github.com/marten-seemann/qtls-go1-16 v0.1.4 // indirect
	github.com/marten-seemann/qtls-go1-17 v0.1.0 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/pires/go-proxyproto v0.6.1 // indirect
	github.com/riobard/go-bloom v0.0.0-20200614022211-cdc8013cb5b3 // indirect
	github.com/seiflotfy/cuckoofilter v0.0.0-20201222105146-bc6005554a0c // indirect
	github.com/v2fly/BrowserBridge v0.0.0-20210430233438-0570fc1d7d08 // indirect
	github.com/v2fly/ss-bloomring v0.0.0-20210312155135-28617310f63e // indirect
	github.com/xtaci/smux v1.5.16 // indirect
	go.starlark.net v0.0.0-20211013185944-b0039bd2cfe3 // indirect
	go4.org/intern v0.0.0-20210108033219-3eb7198706b2 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20201222180813-1025295fd063 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/net v0.0.0-20211020060615-d418f374d309 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/tools v0.1.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto v0.0.0-20211021150943-2b146023228c // indirect
	google.golang.org/grpc v1.41.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	inet.af/netaddr v0.0.0-20210903134321-85fa6c94624e // indirect
)
