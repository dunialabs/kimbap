package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/approvals"
	"github.com/dunialabs/kimbap-core/internal/audit"
	"github.com/dunialabs/kimbap-core/internal/store"
)

type storeApprovalExpirer struct{ st *store.SQLStore }
func (e *storeApprovalExpirer) ExpireStale(ctx context.Context) (int, error) {
	if e==nil||e.st==nil { return 0,nil }
	return e.st.ExpirePendingApprovals(ctx)
}

type storeAuditWriter struct{ st *store.SQLStore }
func (w *storeAuditWriter) Write(ctx context.Context, event audit.AuditEvent) error {
	if w==nil||w.st==nil { return nil }
	errCode,errMsg:="",""
	if event.Error!=nil { errCode=event.Error.Code; errMsg=event.Error.Message }
	metaJSON,inputJSON:="{}","{}"
	if event.Meta!=nil { if b,e:=json.Marshal(event.Meta);e==nil { metaJSON=string(b) } }
	if event.Input!=nil { if b,e:=json.Marshal(event.Input);e==nil { inputJSON=string(b) } }
	return w.st.WriteAuditEvent(ctx,&store.AuditRecord{
		Timestamp:event.Timestamp,RequestID:event.RequestID,TraceID:event.TraceID,
		TenantID:event.TenantID,PrincipalID:event.PrincipalID,AgentName:event.AgentName,
		Service:event.Service,Action:event.Action,Mode:event.Mode,
		Status:string(event.Status),PolicyDecision:event.PolicyDecision,DurationMS:event.DurationMS,
		ErrorCode:errCode,ErrorMessage:errMsg,InputJSON:inputJSON,MetaJSON:metaJSON,
	})
}
func (w *storeAuditWriter) Close() error { return nil }

type storeApprovalStoreAdapter struct{ st *store.SQLStore }
func (a *storeApprovalStoreAdapter) Create(ctx context.Context,req *approvals.ApprovalRequest) error {
	if a.st==nil { return fmt.Errorf("store unavailable") }
	inputJSON:="{}"
	if req.Input!=nil { if b,e:=json.Marshal(req.Input);e==nil { inputJSON=string(b) } }
	return a.st.CreateApproval(ctx,&store.ApprovalRecord{
		ID:req.ID,TenantID:req.TenantID,RequestID:req.RequestID,AgentName:req.AgentName,
		Service:req.Service,Action:req.Action,Status:string(req.Status),
		InputJSON:inputJSON,CreatedAt:req.CreatedAt,ExpiresAt:req.ExpiresAt,
	})
}
func (a *storeApprovalStoreAdapter) Get(ctx context.Context,id string) (*approvals.ApprovalRequest,error) {
	if a.st==nil { return nil,fmt.Errorf("store unavailable") }
	rec,err:=a.st.GetApproval(ctx,id)
	if err!=nil { return nil,err }
	res:=&approvals.ApprovalRequest{
		ID:rec.ID,TenantID:rec.TenantID,RequestID:rec.RequestID,AgentName:rec.AgentName,
		Service:rec.Service,Action:rec.Action,Status:approvals.ApprovalStatus(rec.Status),
		CreatedAt:rec.CreatedAt,ExpiresAt:rec.ExpiresAt,ResolvedBy:rec.ResolvedBy,DenyReason:rec.Reason,
	}
	if rec.ResolvedAt!=nil { res.ResolvedAt=rec.ResolvedAt }
	return res,nil
}
func (a *storeApprovalStoreAdapter) Update(ctx context.Context,req *approvals.ApprovalRequest) error {
	if a.st==nil { return fmt.Errorf("store unavailable") }
	return a.st.UpdateApprovalStatus(ctx,req.ID,string(req.Status),req.ResolvedBy,req.DenyReason)
}
func (a *storeApprovalStoreAdapter) ListPending(ctx context.Context,tenantID string) ([]approvals.ApprovalRequest,error) {
	if a.st==nil { return nil,fmt.Errorf("store unavailable") }
	recs,err:=a.st.ListApprovals(ctx,tenantID,"pending")
	if err!=nil { return nil,err }
	out:=make([]approvals.ApprovalRequest,len(recs))
	for i,r:=range recs { out[i]=approvals.ApprovalRequest{ID:r.ID,TenantID:r.TenantID,RequestID:r.RequestID,AgentName:r.AgentName,Service:r.Service,Action:r.Action,Status:approvals.ApprovalStatus(r.Status)} }
	return out,nil
}
func (a *storeApprovalStoreAdapter) ListAll(ctx context.Context,tenantID string,filter approvals.ApprovalFilter) ([]approvals.ApprovalRequest,error) {
	if a.st==nil { return nil,fmt.Errorf("store unavailable") }
	status:=""
	if filter.Status!=nil { status=string(*filter.Status) }
	recs,err:=a.st.ListApprovals(ctx,tenantID,status)
	if err!=nil { return nil,err }
	out:=make([]approvals.ApprovalRequest,len(recs))
	for i,r:=range recs { out[i]=approvals.ApprovalRequest{ID:r.ID,TenantID:r.TenantID,RequestID:r.RequestID,AgentName:r.AgentName,Service:r.Service,Action:r.Action,Status:approvals.ApprovalStatus(r.Status)} }
	return out,nil
}
func (a *storeApprovalStoreAdapter) ExpireOld(ctx context.Context) (int,error) {
	if a.st==nil { return 0,fmt.Errorf("store unavailable") }
	return a.st.ExpirePendingApprovals(ctx)
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw)==""{ return nil }
	parts:=strings.Split(raw,",")
	out:=make([]string,0,len(parts))
	for _,p:=range parts { if t:=strings.TrimSpace(p);t!=""{ out=append(out,t) } }
	return out
}
