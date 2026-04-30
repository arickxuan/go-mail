package handlers

import (
	"fmt"
	"io"
	"mail0/util"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetVisitorIP(c *gin.Context) {
	ip := getRealIP(c)
	ipv4, ipv6 := splitIP(ip)
	fip, ips := getForwardedIP(c)
	fipv4, fipv6 := splitIP(fip)
	host := getHostIP(c)

	//w.Header().Set("Content-Type", "application/json")
	c.Set("Content-Type", "application/json")
	//fmt.Fprintf(w, `{"ip": "%s", "ipv4": "%s", "ipv6": "%s", "Forwarde ipv4": "%s", "Forwarde ipv6": "%s", "Forwarde ips": "%s", "host": "%s"}`, ip, ipv4, ipv6, fipv4, fipv6, ips, host)
	c.JSON(http.StatusOK, gin.H{"ip": ip, "ipv4": ipv4, "ipv6": ipv6, "Forwarde ipv4": fipv4, "Forwarde ipv6": fipv6, "Forwarde ips": ips, "host": host})
}

func (h *Handler) GetIPHtml(c *gin.Context) {
	html, err := os.ReadFile("public/ip.html")
	if err != nil {
		infos1, _ := util.ListDirectory("./")
		infos, _ := util.ListDirectory("./dir")

		//c.JSON(http.StatusOK, gin.H{"files": infos})
		c.JSON(http.StatusInternalServerError, gin.H{"files": infos, "files1": infos1, "error": "read file: " + err.Error()})
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", html)

}

// proxyIPWho 服务端请求 ipwho.is，避免浏览器直连第三方时的 CORS 限制。
func ProxyIPWho(c *gin.Context) {
	//w.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Set("Content-Type", "application/json; charset=utf-8")
	if c.Request.Method != http.MethodGet {
		//w.WriteHeader(http.StatusMethodNotAllowed)
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
		//_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}
	ipStr := strings.TrimSpace(c.Request.URL.Query().Get("ip"))
	if net.ParseIP(ipStr) == nil {
		//w.WriteHeader(http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ip"})
		//_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid ip"})
		return
	}
	target := "https://ipwho.is/" + url.PathEscape(ipStr)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(target)
	if err != nil {
		//w.WriteHeader(http.StatusBadGateway)
		//_ = json.NewEncoder(w).Encode(map[string]string{"error": "upstream: " + err.Error()})
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		//w.WriteHeader(http.StatusBadGateway)
		//_ = json.NewEncoder(w).Encode(map[string]string{"error": "read upstream: " + err.Error()})
		c.JSON(http.StatusBadGateway, gin.H{"error": "read upstream: " + err.Error()})
		return
	}
	if resp.StatusCode != http.StatusOK {
		//w.WriteHeader(http.StatusBadGateway)
		//_ = json.NewEncoder(w).Encode(map[string]string{"error": "upstream status " + resp.Status})
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream status " + resp.Status})
		//_ = json.NewEncoder(w).Encode(map[string]string{"error": "upstream status " + resp.Status})
		return
	}
	//w.WriteHeader(http.StatusOK)
	//_, _ = w.Write(body)
	c.JSON(http.StatusOK, gin.H{"data": string(body)})
}

func getForwardedIP(c *gin.Context) (string, string) {
	// 检查 X-Forwarded-For
	xForwardedFor := c.Request.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		fmt.Println(ips)
		rawIP := strings.TrimSpace(ips[0])
		if net.ParseIP(rawIP) != nil {
			return rawIP, xForwardedFor
		}
	}
	return "", ""
}

func getRealIP(c *gin.Context) string {

	// 检查 X-Real-IP
	xRealIP := c.Request.Header.Get("X-Real-IP")
	if xRealIP != "" {
		if net.ParseIP(xRealIP) != nil {
			return xRealIP
		}
	}

	return ""

}

func getHostIP(c *gin.Context) string {
	// 从 RemoteAddr 获取
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

func splitIP(ipStr string) (ipv4, ipv6 string) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", ""
	}

	if ip.To4() != nil {
		return ipStr, ""
	}

	if ip.To16() != nil {
		return "", ipStr
	}

	return "", ""
}
