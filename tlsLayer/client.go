package tlsLayer

import (
	"crypto/tls"
	"net"
	"unsafe"

	"github.com/e1732a364fed/v2ray_simple/utils"
	utls "github.com/refraction-networking/utls"
	"go.uber.org/zap"
)

// 关于utls的简单分析，可参考
//https://github.com/e1732a364fed/v2ray_simple/discussions/7

type Client struct {
	tlsConfig  *tls.Config
	uTlsConfig utls.Config
	use_uTls   bool
	alpnList   []string
}

func NewClient(conf Conf) *Client {

	c := &Client{
		use_uTls: conf.Use_uTls,
	}

	c.alpnList = conf.AlpnList

	if conf.Use_uTls {

		c.uTlsConfig = GetUTlsConfig(conf)

		if ce := utils.CanLogInfo("Using uTls and Chrome fingerprint for"); ce != nil {
			ce.Write(zap.String("host", conf.Host))
		}
	} else {

		c.tlsConfig = GetTlsConfig(false, conf)

	}

	return c
}

func (c *Client) Handshake(underlay net.Conn) (tlsConn *Conn, err error) {

	if c.use_uTls {
		configCopy := c.uTlsConfig //发现uTlsConfig竟然没法使用指针，握手一次后配置文件就会被污染，只能拷贝
		//否则的话接下来的握手客户端会报错： tls: CurvePreferences includes unsupported curve

		utlsConn := utls.UClient(underlay, &configCopy, utls.HelloChrome_Auto)
		err = utlsConn.Handshake()
		if err != nil {
			return
		}
		tlsConn = &Conn{
			Conn:           utlsConn,
			ptr:            unsafe.Pointer(utlsConn.Conn),
			tlsPackageType: utlsPackage,
		}

	} else {
		officialConn := tls.Client(underlay, c.tlsConfig)
		err = officialConn.Handshake()
		if err != nil {
			return
		}

		tlsConn = &Conn{
			Conn:           officialConn,
			ptr:            unsafe.Pointer(officialConn),
			tlsPackageType: official,
		}

	}
	return
}
