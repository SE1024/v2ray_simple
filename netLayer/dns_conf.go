package netLayer

import (
	"log"
	"net"
	"net/netip"

	"github.com/hahahrfool/v2ray_simple/utils"
	"go.uber.org/zap"
)

type DnsConf struct {
	Strategy int64          `toml:"strategy"` //0表示默认(和4含义相同), 4表示先查ip4后查ip6, 6表示先查6后查4; 40表示只查ipv4, 60 表示只查ipv6
	Hosts    map[string]any `toml:"hosts"`    //用于强制指定哪些域名会被解析为哪些具体的ip；可以为一个ip字符串，或者一个 []string 数组, 数组内可以是A,AAAA或CNAME
	Servers  []any          `toml:"servers"`  //可以为一个地址url字符串，或者为 SpecialDnsServerConf; 如果第一个元素是url字符串形式，则此第一个元素将会被用作默认dns服务器
}

type SpecialDnsServerConf struct {
	AddrUrlStr string   `toml:"addr"`   //必须为 udp://1.1.1.1:53 这种格式
	Domains    []string `toml:"domain"` //指定哪些域名需要通过 该dns服务器进行查询
}

func loadSpecialDnsServerConf_fromTomlUnmarshaledMap(m map[string]any) *SpecialDnsServerConf {
	addr := m["addr"]
	if addr == nil {
		return nil
	}
	addrStr, ok := addr.(string)
	if !ok {
		return nil
	}
	domains := m["domain"]
	if domains == nil {
		return nil
	}
	domainsAnySlice, ok := domains.([]any)
	if !ok {
		return nil
	}
	domainsSlice := []string{}

	for _, anyD := range domainsAnySlice {
		dstr, ok := anyD.(string)
		if !ok {
			return nil
		}
		domainsSlice = append(domainsSlice, dstr)
	}
	return &SpecialDnsServerConf{
		Domains:    domainsSlice,
		AddrUrlStr: addrStr,
	}

}

func LoadDnsMachine(conf *DnsConf) *DNSMachine {
	var dm = &DNSMachine{TypeStrategy: conf.Strategy}

	var ok = false

	if len(conf.Servers) > 0 {
		//log.Println("conf.Servers", conf.Servers)
		ok = true
		servers := conf.Servers

		dm.SpecialServerPollicy = make(map[string]string)

		for _, ser := range servers {
			switch server := ser.(type) {
			case string:
				ad, e := NewAddrByURL(server)
				if e != nil {
					continue
				}

				dm.AddNewServer(server, &ad)

			case map[string]any:

				realServer := loadSpecialDnsServerConf_fromTomlUnmarshaledMap(server)
				if realServer == nil {
					continue
				}

				if len(realServer.Domains) <= 0 { //既然是特殊dns服务器, 那么就必须指定哪些域名要使用该dns服务器进行查询
					continue
				}

				addr, e := NewAddrByURL(realServer.AddrUrlStr)
				if e != nil {
					continue
				}

				if err := dm.AddNewServer(realServer.AddrUrlStr, &addr); err != nil {
					continue
				}

				for _, thisdomain := range realServer.Domains {
					dm.SpecialServerPollicy[thisdomain] = realServer.AddrUrlStr
				}

			}
		}

	}
	if conf.Hosts != nil {
		ok = true
		dm.SpecialIPPollicy = make(map[string][]netip.Addr)

		for thishost, things := range conf.Hosts {

			switch value := things.(type) {
			case string:
				ip := net.ParseIP(value)

				ad, _ := netip.AddrFromSlice(ip)

				dm.SpecialIPPollicy[thishost] = []netip.Addr{ad}

			case []string:
				for _, str := range value {
					ad, err := NewAddrFromAny(str)
					if err != nil {
						if utils.ZapLogger != nil {
							utils.ZapLogger.Fatal("LoadDnsMachine loading host err", zap.Error(err))
						} else {
							log.Fatalf("LoadDnsMachine loading host err %s\n", err)

						}
					}

					dm.SpecialIPPollicy[thishost] = append(dm.SpecialIPPollicy[thishost], ad.GetHashable().Addr())
				}
			}

		}
	}

	if !ok {
		return nil
	}
	return dm
}