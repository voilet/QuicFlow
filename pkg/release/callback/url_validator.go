package callback

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// URLValidator 回调URL验证器
type URLValidator struct {
	// 允许的协议
	AllowedSchemes []string
	// 禁止的IP范围（CIDR格式）
	BlockedCIDRs []*net.IPNet
	// 禁止的主机名
	BlockedHosts []string
	// 是否允许私有IP
	AllowPrivateIPs bool
	// 是否允许localhost
	AllowLocalhost bool
}

// DefaultURLValidator 默认URL验证器
var DefaultURLValidator = &URLValidator{
	AllowedSchemes: []string{"http", "https"},
	BlockedCIDRs:   parseDefaultBlockedCIDRs(),
	BlockedHosts: []string{
		"localhost",
		"localhost.localdomain",
		"127.0.0.1",
		"::1",
		"0.0.0.0",
	},
	AllowPrivateIPs: false,
	AllowLocalhost:  false,
}

// parseDefaultBlockedCIDRs 解析默认禁止的CIDR范围
func parseDefaultBlockedCIDRs() []*net.IPNet {
	// 禁止的私有和特殊IP范围
	cidrStrings := []string{
		"10.0.0.0/8",      // 私有网络
		"172.16.0.0/12",   // 私有网络
		"192.168.0.0/16",  // 私有网络
		"127.0.0.0/8",     // 回环地址
		"169.254.0.0/16",  // 链路本地地址
		"0.0.0.0/8",       // 当前网络
		"224.0.0.0/4",     // 多播地址
		"240.0.0.0/4",     // 保留地址
		"fc00::/7",        // IPv6 唯一本地地址
		"fe80::/10",       // IPv6 链路本地地址
		"::1/128",         // IPv6 回环地址
	}

	var cidrs []*net.IPNet
	for _, cidr := range cidrStrings {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			cidrs = append(cidrs, ipNet)
		}
	}
	return cidrs
}

// ValidationResult URL验证结果
type ValidationResult struct {
	Valid   bool   `json:"valid"`
	Error   string `json:"error,omitempty"`
	URL     string `json:"url,omitempty"`
	Host    string `json:"host,omitempty"`
	Scheme  string `json:"scheme,omitempty"`
	Warning string `json:"warning,omitempty"`
}

// ValidateURL 验证回调URL
func (v *URLValidator) ValidateURL(rawURL string) ValidationResult {
	result := ValidationResult{
		URL: rawURL,
	}

	// 检查空URL
	if strings.TrimSpace(rawURL) == "" {
		result.Error = "URL cannot be empty"
		return result
	}

	// 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		result.Error = fmt.Sprintf("Invalid URL format: %v", err)
		return result
	}

	result.Scheme = parsedURL.Scheme
	result.Host = parsedURL.Hostname()

	// 检查协议
	schemeAllowed := false
	for _, scheme := range v.AllowedSchemes {
		if strings.ToLower(parsedURL.Scheme) == scheme {
			schemeAllowed = true
			break
		}
	}
	if !schemeAllowed {
		result.Error = fmt.Sprintf("Scheme '%s' is not allowed. Allowed schemes: %s",
			parsedURL.Scheme, strings.Join(v.AllowedSchemes, ", "))
		return result
	}

	// 检查主机名
	host := parsedURL.Hostname()
	if host == "" {
		result.Error = "Host is required"
		return result
	}

	// 检查禁止的主机名
	if !v.AllowLocalhost {
		hostLower := strings.ToLower(host)
		for _, blocked := range v.BlockedHosts {
			if hostLower == strings.ToLower(blocked) {
				result.Error = fmt.Sprintf("Host '%s' is not allowed", host)
				return result
			}
		}
	}

	// 解析IP地址
	ips, err := net.LookupIP(host)
	if err != nil {
		// 如果无法解析，可能是无效的主机名
		// 在生产环境中，我们仍然允许它，因为DNS解析可能在回调发送时成功
		result.Warning = fmt.Sprintf("Could not resolve host '%s': %v", host, err)
		result.Valid = true
		return result
	}

	// 检查解析后的IP地址
	if !v.AllowPrivateIPs {
		for _, ip := range ips {
			// 检查是否在禁止的CIDR范围内
			for _, cidr := range v.BlockedCIDRs {
				if cidr.Contains(ip) {
					result.Error = fmt.Sprintf("IP address %s (resolved from %s) is not allowed: private/reserved IP range",
						ip.String(), host)
					return result
				}
			}

			// 额外检查私有IP
			if ip.IsPrivate() {
				result.Error = fmt.Sprintf("IP address %s (resolved from %s) is a private IP address",
					ip.String(), host)
				return result
			}

			// 检查回环地址
			if ip.IsLoopback() {
				result.Error = fmt.Sprintf("IP address %s (resolved from %s) is a loopback address",
					ip.String(), host)
				return result
			}

			// 检查链路本地地址
			if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				result.Error = fmt.Sprintf("IP address %s (resolved from %s) is a link-local address",
					ip.String(), host)
				return result
			}
		}
	}

	result.Valid = true
	return result
}

// ValidateCallbackURL 便捷函数：使用默认验证器验证URL
func ValidateCallbackURL(rawURL string) ValidationResult {
	return DefaultURLValidator.ValidateURL(rawURL)
}

// NewURLValidator 创建自定义URL验证器
func NewURLValidator(allowPrivateIPs, allowLocalhost bool) *URLValidator {
	return &URLValidator{
		AllowedSchemes:  []string{"http", "https"},
		BlockedCIDRs:    parseDefaultBlockedCIDRs(),
		BlockedHosts:    DefaultURLValidator.BlockedHosts,
		AllowPrivateIPs: allowPrivateIPs,
		AllowLocalhost:  allowLocalhost,
	}
}

// MustValidateURL 验证URL，如果无效则返回错误
func MustValidateURL(rawURL string) error {
	result := ValidateCallbackURL(rawURL)
	if !result.Valid {
		return fmt.Errorf(result.Error)
	}
	return nil
}
