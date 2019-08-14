module github.com/ccbrown/api-fu/examples/chat

go 1.12

require (
	github.com/ccbrown/api-fu v0.0.0
	github.com/ccbrown/go-immutable v0.0.0-20190717001318-30093e84971b // indirect
	github.com/ccbrown/keyvaluestore v0.0.0-20190807034003-d24ee7ef9cb1
	github.com/gobuffalo/packr/v2 v2.5.3-0.20190708182234-662c20c19dde
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.3.0
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20190724013045-ca1201d0de80 // indirect
	golang.org/x/sys v0.0.0-20190804053845-51ab0e2deafa // indirect
	google.golang.org/appengine v1.6.1 // indirect
)

replace github.com/ccbrown/api-fu => ../../
