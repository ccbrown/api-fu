module github.com/ccbrown/api-fu/examples/chat

go 1.12

require (
	github.com/ccbrown/api-fu v0.0.0
	github.com/ccbrown/keyvaluestore v0.0.0-20190807034003-d24ee7ef9cb1
	github.com/go-redis/redis v6.15.3+incompatible
	github.com/gobuffalo/packr/v2 v2.5.3-0.20190708182234-662c20c19dde
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/playcall-inc/backend v0.0.0-20190813193547-260fc7337e6a // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.3.1-0.20190712000136-221dbe5ed467
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
)

replace github.com/ccbrown/api-fu => ../../
