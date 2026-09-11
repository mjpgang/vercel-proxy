package api

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"golang.org/x/net/html"
)

const (
	githubRepoURL     = "https://github.com/tbxark/vercel-proxy"
	identityEncoding  = "identity"
	proxyTargetCookie = "__koyota_proxy_target"

	corsAllowOrigin  = "*"
	corsAllowMethods = "POST, GET, OPTIONS, PUT, DELETE"
	corsAllowHeaders = "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-PROXY-HOST, X-PROXY-SCHEME"
)

var (
	proxyURLPattern = regexp.MustCompile(`^/*(https?:)/*`)
	defaultProxy    = mustNewProxy(Config{})
)

const indexHTML = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="theme-color" content="#000">
<title>Koyota HUB</title>

<style>
:root{
	--bg:#000;
	--panel:#080808;
	--line:#252525;
	--text:#f4f4f4;
	--muted:#777;
}

*{
	box-sizing:border-box;
	margin:0;
	padding:0;
}

html,body{
	width:100%;
	min-height:100%;
	background:#000;
	color:var(--text);
	font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Arial,sans-serif;
}

body{
	min-height:100vh;
	display:flex;
	justify-content:center;
}

button,input{
	font:inherit;
}

a{
	color:inherit;
	text-decoration:none;
}

.wrap{
	width:min(1080px,100%);
	padding:28px 22px 70px;
}

.top{
	display:flex;
	align-items:center;
	justify-content:space-between;
	padding-bottom:20px;
	border-bottom:1px solid var(--line);
}

.logo{
	font-size:18px;
	font-weight:700;
	letter-spacing:-.5px;
}

.logo span{
	color:#777;
	font-weight:400;
}

.status{
	display:flex;
	align-items:center;
	gap:8px;
	color:#777;
	font-size:12px;
}

.dot{
	width:7px;
	height:7px;
	border-radius:50%;
	background:#fff;
	box-shadow:0 0 10px rgba(255,255,255,.5);
}

.hero{
	padding:90px 0 70px;
}

.hero small{
	color:#777;
	font-size:12px;
	letter-spacing:2px;
}

.hero h1{
	margin-top:16px;
	font-size:clamp(46px,9vw,96px);
	line-height:.9;
	letter-spacing:-6px;
	font-weight:800;
}

.hero p{
	margin-top:28px;
	max-width:620px;
	color:#888;
	font-size:15px;
	line-height:1.8;
}

.box{
	margin-top:45px;
	border:1px solid var(--line);
	background:#080808;
}

.input-row{
	display:flex;
	align-items:stretch;
}

.input{
	flex:1;
	min-width:0;
	padding:20px;
	border:0;
	outline:0;
	background:transparent;
	color:#fff;
	font-size:15px;
}

.input::placeholder{
	color:#555;
}

.open{
	border:0;
	border-left:1px solid var(--line);
	background:#fff;
	color:#000;
	padding:0 28px;
	font-weight:700;
	cursor:pointer;
}

.open:hover{
	background:#ddd;
}

.open:disabled{
	opacity:.5;
	cursor:wait;
}

.message{
	padding:0 20px 16px;
	color:#777;
	font-size:12px;
}

.error{
	color:#ff6666;
}

.section{
	margin-top:20px;
	border-top:1px solid var(--line);
}

.section-title{
	padding:18px 0;
	color:#666;
	font-size:11px;
	letter-spacing:2px;
	text-transform:uppercase;
}

.row{
	display:flex;
	align-items:center;
	justify-content:space-between;
	gap:20px;
	padding:19px 0;
	border-top:1px solid var(--line);
}

.row-left{
	display:flex;
	flex-direction:column;
	gap:5px;
}

.row-name{
	font-size:14px;
}

.row-desc{
	color:#666;
	font-size:12px;
}

.row-link{
	color:#aaa;
	font-size:12px;
}

.row-link:hover{
	color:#fff;
}

.footer{
	margin-top:80px;
	padding-top:20px;
	border-top:1px solid var(--line);
	display:flex;
	justify-content:space-between;
	gap:20px;
	color:#555;
	font-size:11px;
}

@media(max-width:600px){
	.wrap{
		padding:20px 16px 50px;
	}

	.hero{
		padding:65px 0 50px;
	}

	.hero h1{
		letter-spacing:-3px;
	}

	.input-row{
		flex-direction:column;
	}

	.open{
		border-left:0;
		border-top:1px solid var(--line);
		padding:17px;
	}

	.footer{
		flex-direction:column;
	}
}
</style>
</head>

<body>

<div class="wrap">

<header class="top">
	<div class="logo">
		Koyota <span>HUB.</span>
	</div>

	<div class="status">
		<span class="dot"></span>
		<span>ONLINE</span>
	</div>
</header>

<main>

<section class="hero">

	<small>WEB PROXY</small>

	<h1>Koyota<br>HUB.</h1>

	<p>
		Enter a URL below and open it through the proxy.
		A direct proxy URL will be generated automatically.
	</p>

	<div class="box">

		<div class="input-row">

			<input
				id="url"
				class="input"
				type="text"
				placeholder="https://example.com"
				autocomplete="off"
				spellcheck="false"
			>

			<button id="open" class="open">
				OPEN
			</button>

		</div>

		<div id="message" class="message">
			Ready.
		</div>

	</div>

</section>

<section class="section">

	<div class="section-title">
		Quick access
	</div>

	<div class="row">

		<div class="row-left">
			<div class="row-name">
				Example
			</div>

			<div class="row-desc">
				Open example.com through the proxy
			</div>
		</div>

		<a
			class="row-link"
			href="/https://example.com"
		>
			OPEN →
		</a>

	</div>

	<div class="row">

		<div class="row-left">
			<div class="row-name">
				Direct proxy
			</div>

			<div class="row-desc">
				Direct proxy URLs remain supported
			</div>
		</div>

		<div class="row-link">
			/https://...
		</div>

	</div>

</section>

</main>

<footer class="footer">
	<span>Koyota HUB</span>
	<span>Powered by vercel-proxy</span>
</footer>

</div>

<script>
const input = document.getElementById("url");
const button = document.getElementById("open");
const message = document.getElementById("message");

function status(text, error){
	message.textContent = text;
	message.className = error
		? "message error"
		: "message";
}

function normalizeURL(value){
	value = value.trim();

	if(!value){
		throw new Error("URLを入力してください");
	}

	if(!/^https?:\/\//i.test(value)){
		value = "https://" + value;
	}

	const u = new URL(value);

	if(
		u.protocol !== "http:" &&
		u.protocol !== "https:"
	){
		throw new Error("HTTP / HTTPS のURLのみ使用できます");
	}

	return u.href;
}

async function openProxy(){

	try{

		const target = normalizeURL(input.value);

		button.disabled = true;
		button.textContent = "CREATING";

		status(
			"Opening proxy URL...",
			false
		);

		const response = await fetch(
			"/create?url=" +
			encodeURIComponent(target),
			{
				method:"GET",
				cache:"no-store"
			}
		);

		if(!response.ok){
			const text = await response.text();
			throw new Error(
				text || "Failed to create proxy URL"
			);
		}

		const data = await response.json();

		if(!data.url){
			throw new Error(
				"Invalid server response"
			);
		}

		status("Opening...",false);

		window.location.href = data.url;

	}catch(error){

		status(
			error.message ||
			"Failed to create proxy URL",
			true
		);

		button.disabled = false;
		button.textContent = "OPEN";
	}
}

button.addEventListener(
	"click",
	openProxy
);

input.addEventListener(
	"keydown",
	function(e){
		if(e.key === "Enter"){
			openProxy();
		}
	}
);
</script>

</body>
</html>`

type Config struct {
	Socks5Proxy        string   `json:"socks5Proxy,omitempty"`
	DomainWhitelist    []string `json:"domainWhitelist,omitempty"`
	DisableCompression bool     `json:"disableCompression,omitempty"`
	DisableGlobalCORS  bool     `json:"disableGlobalCors,omitempty"`
}

type Proxy struct {
	client             *http.Client
	domainWhitelist    []domainRule
	disableCompression bool
	globalCORS         bool
}

func NewProxy(config Config) (*Proxy, error) {

	client, err := newHTTPClient(
		config.Socks5Proxy,
	)

	if err != nil {
		return nil, err
	}

	proxy := &Proxy{
		client: client,
		domainWhitelist: normalizeDomainWhitelist(
			config.DomainWhitelist,
		),
		disableCompression: config.DisableCompression,
		globalCORS:         !config.DisableGlobalCORS,
	}

	proxy.client.CheckRedirect =
		proxy.checkRedirect

	return proxy, nil
}

func mustNewProxy(config Config) *Proxy {

	proxy, err := NewProxy(config)

	if err != nil {
		panic(err)
	}

	return proxy
}

func Handler(
	w http.ResponseWriter,
	r *http.Request,
) {
	defaultProxy.ServeHTTP(w, r)
}

func (p *Proxy) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {

	defer func() {

		if err := recover(); err != nil {

			log.Printf(
				"WithHandler panic: %v",
				err,
			)

			http.Error(
				w,
				fmt.Sprintf(
					"internal server error: %v",
					err,
				),
				http.StatusInternalServerError,
			)
		}

	}()

	if p.globalCORS {

		setCORSHeaders(w)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	/*
		TOP
	*/

	if r.URL.Path == "/" {

		w.Header().Set(
			"Content-Type",
			"text/html; charset=utf-8",
		)

		w.Header().Set(
			"Cache-Control",
			"no-store",
		)

		w.WriteHeader(http.StatusOK)

		_, _ = io.WriteString(
			w,
			indexHTML,
		)

		return
	}

	/*
		CREATE PROXY URL

		/create?url=https://example.com
	*/

	if r.URL.Path == "/create" {

		p.handleCreate(
			w,
			r,
		)

		return
	}

	/*
		LEGACY

		/https://example.com
	*/

	rawURL := proxyURL(r)

	targetURL, err :=
		parseTargetURL(rawURL)

	if err != nil {
		if cookie, cookieErr := r.Cookie(proxyTargetCookie); cookieErr == nil {
			if savedTarget, decodeErr := url.QueryUnescape(cookie.Value); decodeErr == nil {
				if baseURL, baseErr := parseTargetURL(savedTarget); baseErr == nil {
					targetURL = baseURL.ResolveReference(r.URL)
					err = nil
				}
			}
		}
	}

	if err != nil {

		http.Error(
			w,
			"invalid url: "+rawURL,
			http.StatusBadRequest,
		)

		return
	}

	if err := p.checkDomain(targetURL); err != nil {

		http.Error(
			w,
			err.Error(),
			http.StatusForbidden,
		)

		return
	}

	p.proxyRequest(
		w,
		r,
		targetURL,
		"",
	)
}

func (p *Proxy) handleCreate(
	w http.ResponseWriter,
	r *http.Request,
) {

	if r.Method != http.MethodGet {

		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)

		return
	}

	rawTarget :=
		strings.TrimSpace(
			r.URL.Query().Get("url"),
		)

	if rawTarget == "" {

		http.Error(
			w,
			"missing url",
			http.StatusBadRequest,
		)

		return
	}

	targetURL, err :=
		parseTargetURL(rawTarget)

	if err != nil {

		http.Error(
			w,
			"invalid url",
			http.StatusBadRequest,
		)

		return
	}

	if err := p.checkDomain(targetURL); err != nil {

		http.Error(
			w,
			err.Error(),
			http.StatusForbidden,
		)

		return
	}

	response := map[string]string{
		"url": buildLegacyProxyURL(targetURL),
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(
		response,
	)
}

func (p *Proxy) proxyRequest(
	w http.ResponseWriter,
	r *http.Request,
	targetURL *url.URL,
	randomID string,
) {

	req, err :=
		http.NewRequestWithContext(
			r.Context(),
			r.Method,
			targetURL.String(),
			r.Body,
		)

	if err != nil {

		internalServerError(
			w,
			err,
		)

		return
	}

	copyHeaders(
		r.Header,
		req.Header,
	)
	removeProxyTargetCookie(req.Header)

	if p.disableCompression {

		disableUpstreamCompression(
			req.Header,
		)
	}

	resp, err :=
		p.client.Do(req)

	if err != nil {

		var domainErr *domainNotAllowedError

		if errors.As(
			err,
			&domainErr,
		) {

			http.Error(
				w,
				domainErr.Error(),
				http.StatusForbidden,
			)

			return
		}

		internalServerError(
			w,
			err,
		)

		return
	}

	defer closeResponseBody(resp)

	if err := proxyRaw(
		w,
		resp,
		r,
		p.globalCORS,
		targetURL,
		randomID,
	); err != nil {

		log.Printf(
			"Proxy response error: %v",
			err,
		)
	}
}

func internalServerError(
	w http.ResponseWriter,
	err error,
) {

	if err != nil {

		log.Printf(
			"Internal server error: %v",
			err,
		)

		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
	}
}

/*
	proxyRaw

	ここが今回の重要部分。

	Google等:

	Location: /search?q=wtf

	を

	/randomID/search?q=wtf

	へ変換する。
*/

func proxyRaw(
	w http.ResponseWriter,
	resp *http.Response,
	req *http.Request,
	globalCORS bool,
	currentTarget *url.URL,
	randomID string,
) error {

	copyHeaders(
		resp.Header,
		w.Header(),
	)

	if globalCORS {
		setCORSHeaders(w)
	}

	w.Header().Add(
		"Set-Cookie",
		proxyTargetCookie+"="+url.QueryEscape(currentTarget.String())+"; Path=/; HttpOnly; SameSite=Lax",
	)

	cookies := w.Header().Values("Set-Cookie")
	w.Header().Del("Set-Cookie")
	for _, cookie := range cookies {
		w.Header().Add("Set-Cookie", rewriteSetCookie(cookie, randomID))
	}

	if location :=
		resp.Header.Get("Location"); location != "" {

		locationURL, err :=
			url.Parse(location)

		if err == nil {

			resolvedURL :=
				currentTarget.ResolveReference(
					locationURL,
				)

			if resolvedURL.String() == currentTarget.String() {
				w.Header().Del("Location")
			} else if randomID != "" {

				/*
					ランダムURLなら、
					リダイレクト先も同じIDを使う。
				*/

				proxyLocation :=
					buildRandomProxyURL(
						randomID,
						resolvedURL,
					)

				w.Header().Set(
					"Location",
					proxyLocation,
				)

			} else {

				/*
					従来形式の場合
					/https://... へ戻す。
				*/

				proxyLocation :=
					buildLegacyProxyURL(
						resolvedURL,
					)

				w.Header().Set(
					"Location",
					proxyLocation,
				)
			}
		}
	}

	if refresh := resp.Header.Get("Refresh"); refresh != "" {
		if rewritten, ok := rewriteRefresh(refresh, currentTarget, randomID); ok {
			w.Header().Set("Refresh", rewritten)
		}
	}

	body, transformed, err := rewriteResponseBody(resp, currentTarget, randomID)
	if err != nil {
		return err
	}
	if transformed {
		w.Header().Del("Content-Encoding")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Del("ETag")
		w.Header().Del("Content-MD5")
	}

	if w.Header().Get("Content-Security-Policy") != "" || w.Header().Get("Content-Security-Policy-Report-Only") != "" {
		w.Header().Del("Content-Security-Policy")
		w.Header().Del("Content-Security-Policy-Report-Only")
	}

	if w.Header().Get("Referer") != "" {

		w.Header().Del("Referer")

		w.Header().Add(
			"Referer",
			req.Host,
		)
	}

	w.WriteHeader(
		resp.StatusCode,
	)

	if transformed {
		_, err = w.Write(body)
	} else {
		_, err = io.Copy(w, resp.Body)
	}

	return err
}

func rewriteResponseBody(
	resp *http.Response,
	baseURL *url.URL,
	randomID string,
) ([]byte, bool, error) {
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	isHTML := contentType == "text/html" || contentType == "application/xhtml+xml"
	isCSS := contentType == "text/css"
	if !isHTML && !isCSS {
		return nil, false, nil
	}
	encoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	if encoding != "" && encoding != "gzip" && encoding != "x-gzip" && encoding != "deflate" && encoding != "br" {
		return nil, false, nil
	}

	body, err := decodeResponseBody(resp)
	if err != nil {
		return nil, false, err
	}

	if isHTML {
		doc, err := html.Parse(bytes.NewReader(body))
		if err != nil {
			return nil, false, err
		}
		rewriteHTMLNodes(doc, baseURL, randomID)
		var output bytes.Buffer
		if err := html.Render(&output, doc); err != nil {
			return nil, false, err
		}
		return output.Bytes(), true, nil
	}

	return rewriteCSS(body, baseURL, randomID), true, nil
}

func decodeResponseBody(resp *http.Response) ([]byte, error) {
	encoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	if encoding == "" {
		return io.ReadAll(resp.Body)
	}

	var reader io.Reader
	var closer io.Closer
	switch encoding {
	case "gzip", "x-gzip":
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		reader = gzipReader
		closer = gzipReader
	case "deflate":
		zlibReader, err := zlib.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		reader = zlibReader
		closer = zlibReader
	case "br":
		reader = brotli.NewReader(resp.Body)
	default:
		return io.ReadAll(resp.Body)
	}
	if closer != nil {
		defer closer.Close()
	}
	return io.ReadAll(reader)
}

func rewriteHTMLNodes(node *html.Node, baseURL *url.URL, randomID string) {
	if node.Type == html.TextNode && node.Parent != nil && strings.EqualFold(node.Parent.Data, "style") {
		node.Data = string(rewriteCSS([]byte(node.Data), baseURL, randomID))
	} else if node.Type == html.TextNode && node.Parent != nil && strings.EqualFold(node.Parent.Data, "script") {
		node.Data = rewriteScriptURLs(node.Data, baseURL, randomID)
	}
	if node.Type == html.ElementNode {
		for i := range node.Attr {
			if node.Attr[i].Key == "src" || node.Attr[i].Key == "href" || node.Attr[i].Key == "action" || node.Attr[i].Key == "poster" || node.Attr[i].Key == "cite" {
				node.Attr[i].Val = proxyReference(node.Attr[i].Val, baseURL, randomID)
			} else if node.Attr[i].Key == "srcset" {
				node.Attr[i].Val = rewriteSrcset(node.Attr[i].Val, baseURL, randomID)
			} else if node.Attr[i].Key == "style" {
				node.Attr[i].Val = string(rewriteCSS([]byte(node.Attr[i].Val), baseURL, randomID))
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		rewriteHTMLNodes(child, baseURL, randomID)
	}
}

var scriptRootURLPattern = regexp.MustCompile(`(["'` + "`" + `])(/[^/][^/"'` + "`" + `]*)["'` + "`" + `]`)

func rewriteScriptURLs(value string, baseURL *url.URL, randomID string) string {
	return scriptRootURLPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := scriptRootURLPattern.FindStringSubmatch(match)
		return parts[1] + proxyReference(parts[2], baseURL, randomID) + match[len(match)-1:]
	})
}

func rewriteSrcset(value string, baseURL *url.URL, randomID string) string {
	parts := strings.Split(value, ",")
	for i, part := range parts {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) > 0 {
			fields[0] = proxyReference(fields[0], baseURL, randomID)
			parts[i] = strings.Join(fields, " ")
		}
	}
	return strings.Join(parts, ", ")
}

var cssURLPattern = regexp.MustCompile(`(?is)url\(\s*(["']?)([^"')]+)["']?\s*\)`)
var cssImportPattern = regexp.MustCompile(`(?is)(@import\s+)(["'])([^"']+)(["'])`)

func rewriteCSS(body []byte, baseURL *url.URL, randomID string) []byte {
	text := string(body)
	text = cssURLPattern.ReplaceAllStringFunc(text, func(match string) string {
		submatch := cssURLPattern.FindStringSubmatch(match)
		return "url(" + submatch[1] + proxyReference(submatch[2], baseURL, randomID) + submatch[1] + ")"
	})
	text = cssImportPattern.ReplaceAllStringFunc(text, func(match string) string {
		submatch := cssImportPattern.FindStringSubmatch(match)
		return submatch[1] + submatch[2] + proxyReference(submatch[3], baseURL, randomID) + submatch[4]
	})
	return []byte(text)
}

func proxyReference(reference string, baseURL *url.URL, randomID string) string {
	trimmed := strings.TrimSpace(reference)
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "data" || parsed.Scheme == "javascript" || parsed.Scheme == "mailto" || parsed.Scheme == "tel" || strings.HasPrefix(trimmed, "#") {
		return reference
	}
	resolved := baseURL.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return reference
	}
	if randomID != "" {
		return buildRandomProxyURL(randomID, resolved)
	}
	return buildLegacyProxyURL(resolved)
}

func rewriteRefresh(value string, baseURL *url.URL, randomID string) (string, bool) {
	parts := strings.SplitN(value, ";", 2)
	if len(parts) != 2 {
		return value, false
	}
	fields := strings.SplitN(strings.TrimSpace(parts[1]), "=", 2)
	if len(fields) != 2 || !strings.EqualFold(strings.TrimSpace(fields[0]), "url") {
		return value, false
	}
	reference := strings.Trim(strings.TrimSpace(fields[1]), "\"'")
	return parts[0] + "; url=" + proxyReference(reference, baseURL, randomID), true
}

func rewriteSetCookie(value string, randomID string) string {
	parts := strings.Split(value, ";")
	output := parts[:1]
	for _, part := range parts[1:] {
		trimmed := strings.TrimSpace(part)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "domain=") {
			continue
		}
		if strings.HasPrefix(lower, "path=") && randomID != "" {
			path := strings.TrimSpace(trimmed[len("path="):])
			if !strings.HasPrefix(path, "/"+randomID) {
				path = "/" + randomID + "/" + strings.TrimPrefix(path, "/")
			}
			trimmed = "Path=" + path
		}
		output = append(output, trimmed)
	}
	return strings.Join(output, "; ")
}

func buildRandomProxyURL(
	id string,
	target *url.URL,
) string {

	if target == nil {
		return "/" + id
	}

	path := target.EscapedPath()

	if path == "" {
		path = "/"
	}

	result :=
		"/" + id + path

	if target.RawQuery != "" {

		result +=
			"?" + target.RawQuery
	}

	if target.Fragment != "" {

		result +=
			"#" + target.Fragment
	}

	return result
}

func buildLegacyProxyURL(
	target *url.URL,
) string {

	if target == nil {
		return "/"
	}

	result :=
		"/" +
			target.Scheme +
			"://" +
			target.Host +
			target.EscapedPath()

	if target.RawQuery != "" {

		result +=
			"?" + target.RawQuery
	}

	if target.Fragment != "" {

		result +=
			"#" + target.Fragment
	}

	return result
}

func setCORSHeaders(
	w http.ResponseWriter,
) {

	clearCORSHeaders(
		w.Header(),
	)

	w.Header().Set(
		"Access-Control-Allow-Origin",
		corsAllowOrigin,
	)

	w.Header().Set(
		"Access-Control-Allow-Methods",
		corsAllowMethods,
	)

	w.Header().Set(
		"Access-Control-Allow-Headers",
		corsAllowHeaders,
	)
}

func clearCORSHeaders(
	header http.Header,
) {

	for k := range header {

		if strings.HasPrefix(
			strings.ToLower(k),
			"access-control-",
		) {

			header.Del(k)
		}
	}
}

func proxyURL(
	r *http.Request,
) string {

	u :=
		proxyURLPattern.ReplaceAllString(
			r.URL.Path,
			"$1//",
		)

	if r.URL.RawQuery != "" {

		u +=
			"?" + r.URL.RawQuery
	}

	return u
}

func parseTargetURL(
	rawURL string,
) (*url.URL, error) {

	targetURL, err :=
		url.Parse(rawURL)

	if err != nil {
		return nil, err
	}

	if targetURL.Host == "" ||
		(targetURL.Scheme != "http" &&
			targetURL.Scheme != "https") {

		return nil, fmt.Errorf(
			"unsupported target url: %s",
			rawURL,
		)
	}

	return targetURL, nil
}

func copyHeaders(
	src http.Header,
	dst http.Header,
) {

	for k, v := range src {

		for _, vv := range v {

			dst.Add(k, vv)
		}
	}
}

func removeProxyTargetCookie(header http.Header) {
	values := header.Values("Cookie")
	header.Del("Cookie")
	for _, value := range values {
		cookies := make([]string, 0)
		for _, item := range strings.Split(value, ";") {
			item = strings.TrimSpace(item)
			if item == "" || strings.HasPrefix(item, proxyTargetCookie+"=") {
				continue
			}
			cookies = append(cookies, item)
		}
		if len(cookies) > 0 {
			header.Add("Cookie", strings.Join(cookies, "; "))
		}
	}
}

func disableUpstreamCompression(
	header http.Header,
) {

	header.Set(
		"Accept-Encoding",
		identityEncoding,
	)
}

func closeResponseBody(
	resp *http.Response,
) {

	if err := resp.Body.Close(); err != nil {

		log.Printf(
			"Close response body error: %v",
			err,
		)
	}
}

func newHTTPClient(
	socks5Proxy string,
) (*http.Client, error) {

	transport :=
		http.DefaultTransport.(*http.Transport).
			Clone()

	transport.Proxy = nil

	if strings.TrimSpace(
		socks5Proxy,
	) != "" {

		proxyURL, err :=
			parseSocks5ProxyURL(
				socks5Proxy,
			)

		if err != nil {
			return nil, err
		}

		transport.Proxy =
			http.ProxyURL(proxyURL)
	}

	return &http.Client{
		Transport: transport,
	}, nil
}

func parseSocks5ProxyURL(
	rawProxy string,
) (*url.URL, error) {

	rawProxy =
		strings.TrimSpace(rawProxy)

	if rawProxy == "" {
		return nil, nil
	}

	if !strings.Contains(
		rawProxy,
		"://",
	) {

		rawProxy =
			"socks5://" + rawProxy
	}

	proxyURL, err :=
		url.Parse(rawProxy)

	if err != nil {

		return nil, fmt.Errorf(
			"invalid socks5 proxy: %w",
			err,
		)
	}

	proxyURL.Scheme =
		strings.ToLower(
			proxyURL.Scheme,
		)

	if proxyURL.Scheme != "socks5" &&
		proxyURL.Scheme != "socks5h" {

		return nil, fmt.Errorf(
			"unsupported proxy scheme %q: only socks5 and socks5h are supported",
			proxyURL.Scheme,
		)
	}

	if proxyURL.Host == "" {

		return nil, fmt.Errorf(
			"invalid socks5 proxy: missing host",
		)
	}

	return proxyURL, nil
}

func (p *Proxy) isDomainAllowed(
	targetURL *url.URL,
) bool {

	return isDomainURLAllowed(
		targetURL,
		p.domainWhitelist,
	)
}

func (p *Proxy) checkDomain(
	targetURL *url.URL,
) error {

	if p.isDomainAllowed(targetURL) {
		return nil
	}

	return &domainNotAllowedError{
		host: targetURL.Hostname(),
	}
}

func (p *Proxy) checkRedirect(
	req *http.Request,
	via []*http.Request,
) error {

	if len(via) >= 10 {

		return errors.New(
			"stopped after 10 redirects",
		)
	}

	if err := p.checkDomain(req.URL); err != nil {
		return err
	}

	return http.ErrUseLastResponse
}

type domainNotAllowedError struct {
	host string
}

func (e *domainNotAllowedError) Error() string {

	return "domain not allowed: " +
		e.host
}

type domainRule struct {
	exclude bool
	pattern string
	regexp  *regexp.Regexp
	port    string
}

func isDomainAllowed(
	host string,
	whitelist []domainRule,
) bool {

	target :=
		normalizeTargetHostPort(
			host,
			"",
		)

	return isDomainTargetAllowed(
		target,
		whitelist,
	)
}

func isDomainURLAllowed(
	targetURL *url.URL,
	whitelist []domainRule,
) bool {

	target :=
		normalizeTargetHostPort(
			targetURL.Host,
			targetURL.Scheme,
		)

	return isDomainTargetAllowed(
		target,
		whitelist,
	)
}

func isDomainTargetAllowed(
	target domainTarget,
	whitelist []domainRule,
) bool {

	if len(whitelist) == 0 {
		return true
	}

	allowed := false

	for _, rule := range whitelist {

		if !rule.matches(target) {
			continue
		}

		if rule.exclude {
			return false
		}

		allowed = true
	}

	return allowed
}

func normalizeDomainWhitelist(
	entries []string,
) []domainRule {

	whitelist :=
		make([]domainRule, 0, len(entries))

	seen :=
		make(map[string]struct{}, len(entries))

	for _, entry := range entries {

		rule, ok :=
			normalizeDomainRule(entry)

		if !ok {
			continue
		}

		key := rule.key()

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}

		whitelist =
			append(
				whitelist,
				rule,
			)
	}

	return whitelist
}

func normalizeDomainRule(
	entry string,
) (domainRule, bool) {

	entry =
		strings.TrimSpace(entry)

	exclude :=
		strings.HasPrefix(
			entry,
			"-",
		)

	if exclude {

		entry =
			strings.TrimSpace(
				strings.TrimPrefix(
					entry,
					"-",
				),
			)
	}

	host, port :=
		splitDomainRulePort(entry)

	host =
		normalizeDomain(host)

	if host == "" {
		return domainRule{}, false
	}

	rule := domainRule{
		exclude: exclude,
		pattern: host,
		port:    port,
	}

	if strings.ContainsAny(
		host,
		"*?",
	) {

		rule.regexp =
			compileDomainWildcard(host)
	}

	return rule, true
}

func splitDomainRulePort(
	entry string,
) (string, string) {

	if host, port, err :=
		net.SplitHostPort(entry); err == nil {

		return host,
			normalizePort(port)
	}

	if strings.Count(
		entry,
		":",
	) == 1 {

		idx :=
			strings.LastIndex(
				entry,
				":",
			)

		port :=
			normalizePort(
				entry[idx+1:],
			)

		if port != "" {

			return entry[:idx],
				port
		}
	}

	return entry, ""
}

func normalizePort(
	port string,
) string {

	port =
		strings.TrimSpace(port)

	if port == "" {
		return ""
	}

	for _, r := range port {

		if r < '0' || r > '9' {
			return ""
		}
	}

	return port
}

func compileDomainWildcard(
	pattern string,
) *regexp.Regexp {

	var b strings.Builder

	b.WriteString("^")

	for _, r := range pattern {

		switch r {

		case '*':
			b.WriteString(".*")

		case '?':
			b.WriteString(".")

		default:
			b.WriteString(
				regexp.QuoteMeta(
					string(r),
				),
			)
		}
	}

	b.WriteString("$")

	return regexp.MustCompile(
		b.String(),
	)
}

func (r domainRule) key() string {

	if r.exclude {

		return "-" +
			r.pattern +
			":" +
			r.port
	}

	return r.pattern +
		":" +
		r.port
}

func (r domainRule) matches(
	target domainTarget,
) bool {

	if !r.matchesHost(
		target.host,
	) {

		return false
	}

	return r.matchesPort(
		target.port,
	)
}

func (r domainRule) matchesHost(
	host string,
) bool {

	if r.regexp != nil {

		return r.regexp.MatchString(
			host,
		)
	}

	return host == r.pattern ||
		strings.HasSuffix(
			host,
			"."+r.pattern,
		)
}

func (r domainRule) matchesPort(
	port string,
) bool {

	if r.port == "0" {
		return true
	}

	if r.port != "" {
		return port == r.port
	}

	return port == "" ||
		port == "80" ||
		port == "443"
}

type domainTarget struct {
	host string
	port string
}

func normalizeTargetHostPort(
	rawHost,
	scheme string,
) domainTarget {

	host := rawHost
	port := ""

	if parsedHost, parsedPort, err :=
		net.SplitHostPort(rawHost); err == nil {

		host = parsedHost
		port = parsedPort

	} else if strings.Count(
		rawHost,
		":",
	) == 1 {

		idx :=
			strings.LastIndex(
				rawHost,
				":",
			)

		if parsedPort :=
			normalizePort(
				rawHost[idx+1:],
			); parsedPort != "" {

			host =
				rawHost[:idx]

			port =
				parsedPort
		}
	}

	if port == "" {

		port =
			defaultPort(scheme)
	}

	return domainTarget{
		host: normalizeDomain(host),
		port: port,
	}
}

func defaultPort(
	scheme string,
) string {

	switch strings.ToLower(scheme) {

	case "http":
		return "80"

	case "https":
		return "443"

	default:
		return ""
	}
}

func normalizeDomain(
	domain string,
) string {

	domain =
		strings.Trim(
			strings.ToLower(
				strings.TrimSpace(domain),
			),
			".",
		)

	domain =
		strings.TrimPrefix(
			strings.TrimSuffix(
				domain,
				"]",
			),
			"[",
		)

	if domain == "" {
		return ""
	}

	if host, _, err :=
		net.SplitHostPort(domain); err == nil {

		domain =
			strings.Trim(
				strings.ToLower(host),
				".",
			)
	}

	return domain
}
