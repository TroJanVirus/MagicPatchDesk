package core

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

// SubmitConfidenceReport only creates a NEW reviewable issue after an explicit
// user action. Never used by automatic diagnostics or scheduled sync.
// The repo's reviewed JSON index remains the authoritative downloadable log.
func SubmitConfidenceReport(ctx context.Context,client *http.Client,token string,report ConfidenceReport)(int,error){
    if strings.TrimSpace(token)==""{return 0,errors.New("GitHub Issues token required; no confidence report submitted")}
    body,e:=ConfidenceIssueBody(report)
    if e!=nil{return 0,e}
    if client==nil{client=&http.Client{Timeout:20*time.Second}}
    payload,e:=json.Marshal(map[string]any{
        "title":fmt.Sprintf("[Compatibility] %s %s %s %s",report.TitleID,report.Environment,report.Result,report.PatchSHA256[:12]),
        "body":body,
        "labels":[]string{},
    })
    if e!=nil{return 0,e}
    request,e:=http.NewRequestWithContext(ctx,http.MethodPost,GitHubIssuesAPI+"/issues",bytes.NewReader(payload))
    if e!=nil{return 0,e}
    request.Header.Set("Authorization","Bearer "+token)
    request.Header.Set("Accept","application/vnd.github+json")
    request.Header.Set("Content-Type","application/json")
    request.Header.Set("User-Agent","MagicPatchDesk/"+AppVersion)
    response,e:=client.Do(request)
    if e!=nil{return 0,e}
    defer response.Body.Close()
    if response.StatusCode!=http.StatusCreated{return 0,fmt.Errorf("GitHub refused compatibility report (HTTP %d); no report imported",response.StatusCode)}
    var created struct{Number int `json:"number"`}
    if e=json.NewDecoder(io.LimitReader(response.Body,32<<10)).Decode(&created);e!=nil{return 0,e}
    if created.Number<1{return 0,errors.New("GitHub did not return a confidence issue number")}
    return created.Number,nil
}
