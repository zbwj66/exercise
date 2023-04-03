/*
 * @Descripttion: go_basic
 * @version: 1.0
 * @Author: ZBWJ_CJY
 * @Date: 2023-04-02 22:37:41
 * @LastEditors: ZBWJ_CJY
 * @LastEditTime: 2023-04-03 00:09:48
 */
 package main

 import (
	 "bufio"
	 "crypto/tls"
	 "fmt"
	 "io"
	 "net"
	 "net/http"
	 "net/http/httputil"
 )
 
 var certFile = "pro.pem"
 var keyFile = "pro-key.pem"
 
 func main() {
	 // 建立到B设备的连接
	 conn, err := net.Dial("tcp", "192.168.159.128:9222")
	 if err != nil {
		 panic(err)
	 }
	 defer conn.Close()
 
	 // 监听本地http请求
	 httpln, err := net.Listen("tcp", "127.0.0.1:8855")
	 if err != nil {
		 panic(err)
	 }
	 defer httpln.Close()
	 fmt.Println("Listening on 127.0.0.1:8855")
 
	 // 加载TLS证书
	 cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	 if err != nil {
		 panic(err)
	 }
 
	 // 配置TLS
	 tlsCfg := &tls.Config{
		 Certificates:       []tls.Certificate{cert},
		 InsecureSkipVerify: true,
	 }
 
	 // 监听本地https请求
	 httpsLn, err := tls.Listen("tcp", "127.0.0.1:8854", tlsCfg)
	 if err != nil {
		 panic(err)
	 }
	 defer httpsLn.Close()
	 fmt.Println("Listening on 127.0.0.1:8854")
 
	 // 处理http请求
	 go handleHTTPRequests(httpln, conn)
	 // 处理https请求
	 go handleHTTPSRequests(httpsLn, conn)
	 select {}
 }
 
 // 处理http请求
 func handleHTTPRequests(httpln net.Listener, conn net.Conn) {
	 for {
		 browserConn, err := httpln.Accept()
		 if err != nil {
			 panic(err)
		 }
		 go handleReqResp(browserConn, conn)
	 }
 }
 
 // 处理https请求
 func handleHTTPSRequests(httpsln net.Listener, conn net.Conn) {
	 for {
		 browserConn, err := httpsln.Accept()
		 if err != nil {
			 panic(err)
		 }
		 go handleReqResp(browserConn, conn)
	 }
 }
 
 func handleReqResp(browserConn net.Conn, conn net.Conn) {
	 defer browserConn.Close()
	 // fmt.Printf("Received https request from %s\n",browserConn.RemoteAddr())
 
	 // 将https请求转发到B服务器
	 request, err := http.ReadRequest(bufio.NewReader(browserConn))
	 if err != nil {
		 panic(err)
	 }
 
	 // 将修改后的请求发送给B服务器
	 reqBytes, err := httputil.DumpRequest(request, true)
	 if err != nil {
		 panic(err)
	 }
	 _, err = conn.Write([]byte(reqBytes))
	 if err != nil {
		 panic(err)
	 }
 
	 // 将B设备的响应发送给浏览器
	 buf := make([]byte, 2048)
	 n, err := conn.Read(buf)
	 if err != nil {
		 if err != io.EOF {
			 fmt.Println("Error reading", err.Error())
		 }
		 fmt.Println("err_response:", err)
	 }
 
	 _, err = browserConn.Write(buf[:n])
	 if err != nil {
		 panic(err)
	 }
 }