/*
 * @Descripttion: go_basic
 * @version: 1.0
 * @Author: ZBWJ_CJY
 * @Date: 2023-04-02 20:33:00
 * @LastEditors: ZBWJ_CJY
 * @LastEditTime: 2023-04-03 00:15:59
 */
// Server端
package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
)

var certFile = "pro.pem"
var keyFile = "pro-key.pem"

func main() {
	// 启动HHTTP和HTTPS服务
	go startHTTPServer(":8855")
	go startHTTPSServer(":8854", certFile, keyFile)

	// 监听来自A服务器的TCP连接
	listenTCP(":9222", handleTCPRequest)
}

// 启动HTTP服务
func startHTTPServer(addr string) {
	httpServer := &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(handleRequests),
	}

	err := httpServer.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

// 启动HTTPS连接
func startHTTPSServer(addr string, certFile string, keyFile string) {
	httpsServer := &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(handleRequests),
	}
	err := httpsServer.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		panic(err)
	}
}

// 监听TCP连接
func listenTCP(addr string, handler func(net.Conn)) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("listening on %s\n", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handler(conn)
	}
}

// 处理TCP请求
func handleTCPRequest(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("new client connected: %s\n", conn.RemoteAddr().String())
	for {
		reqBuf := make([]byte, 2048)
		n, err := conn.Read(reqBuf)
		if err != nil {
			fmt.Printf("client %s disconnected: %s\n", conn.RemoteAddr().String())
			break
		}
		// 获取来自A的请求信息
		reqStr := string(reqBuf[:n])
		req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(reqStr)))
		if err != nil {
			fmt.Printf("failed to read request: %s \n", err.Error())
			break
		}

		// 转发请求
		resp, err := forwardRequest(req)
		if err != nil {
			fmt.Printf("failed to forward request: %s\n", err.Error())
			break
		}

		defer resp.Body.Close()

		// 转发响应
		err = sendResponse(conn, resp)
		if err != nil {
			fmt.Printf("failed to send response: %s\n", err.Error())
			break
		}
	}
}

// 转发请求
func forwardRequest(req *http.Request) (*http.Response, error) {
	if req.Host == "127.0.0.1:8854" {
		targetUrl, err := url.Parse("https://127.0.0.1:8854" + req.URL.Path)
		if err != nil {
			panic(err)
		}
		req.URL = targetUrl
		req.RequestURI = ""
	} else if req.Host == "127.0.0.1:8855" {
		targetUrl, err := url.Parse("http://127.0.0.1:8855" + req.URL.Path)
		if err != nil {
			panic(err)
		}
		req.URL = targetUrl
		req.RequestURI = ""
	}

	// 创建HTTP客户端
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// 发送HTTP/HTTPS请求
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("err_resp:", err)
		return nil, err
	}
	return resp, err
}

// 处理web请求
func handleRequests(w http.ResponseWriter, r *http.Request) {
	mac := getMAC()
	if r.TLS != nil {
		fmt.Fprintf(w, "https B mac地址是:%s",mac)
	} else {
		fmt.Fprintf(w, "http B mac地址是:%s",mac)
	}
}

// 获取第一个网卡的MAC地址
func getMAC() string{
	ifaces,err := net.Interfaces()
	if err !=nil{
		panic(err)
	}

	for _,iface := range ifaces{
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0{
			addrs,err := iface.Addrs()
			if err!=nil{
				panic(err)
			}
			for _,addr := range addrs{
				//
				if addr,ok := addr.(*net.IPNet);ok && !addr.IP.IsLoopback(){
					mac := iface.HardwareAddr.String()
					return mac
				}
			}
		}
	}
	return "xxx(unable to obtain mac address)"
}

// 转发响应
func sendResponse(conn net.Conn, resp *http.Response) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("failed to read respBody:", err)
		return err
	}
	// 构建 HTTP 响应并发送给客户端
	r := []byte(fmt.Sprintf("%s %s\r\n", resp.Proto, resp.Status))
	for key, values := range resp.Header {
		for _, value := range values {
			r = append(r, []byte(fmt.Sprintf("%s: %s\r\n", key, value))...)
		}
	}
	r = append(r, []byte("\r\n")...)
	r = append(r, body...)
	r = append(r,[]byte("\r\n")...)
	fmt.Printf("r: %v\n", string(r))
	_, err = conn.Write(r)
	if err != nil {
		fmt.Println("failed to send respBody:", err)
		return err
	}
	return nil
}