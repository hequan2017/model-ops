// 核心业务处理器：安装人员、部署任务、设备验收。
package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hequan2017/model-ops/server/internal/model"
)

var validLevels = map[string]bool{"专家级": true, "高级": true, "中级": true, "初级": true}
var validItemStatus = map[string]bool{
	model.ItemPending: true, model.ItemRunning: true, model.ItemPass: true, model.ItemFail: true, model.ItemNA: true,
}

// ---- 安装人员 ----

func (a *API) listInstallers(w http.ResponseWriter, _ *http.Request) {
	var out []model.Installer
	a.st.View(func(d *model.Data) { out = d.Installers })
	writeJSON(w, http.StatusOK, out)
}

func (a *API) createInstaller(w http.ResponseWriter, r *http.Request) {
	var in model.Installer
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		fail(w, http.StatusBadRequest, "姓名不能为空")
		return
	}
	if !validLevels[in.Level] {
		fail(w, http.StatusBadRequest, "等级必须为：专家级/高级/中级/初级")
		return
	}
	in.Skills = dedup(in.Skills)
	var created *model.Installer
	if err := a.st.Update(func(d *model.Data) error {
		in.ID = d.NextID("INS", 100)
		d.Installers = append(d.Installers, in)
		created = &in
		return nil
	}); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *API) updateInstaller(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in model.Installer
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if !validLevels[in.Level] {
		fail(w, http.StatusBadRequest, "等级必须为：专家级/高级/中级/初级")
		return
	}
	var out *model.Installer
	if err := a.st.Update(func(d *model.Data) error {
		for i := range d.Installers {
			if d.Installers[i].ID == id {
				in.ID = id
				in.Skills = dedup(in.Skills)
				d.Installers[i] = in
				out = &d.Installers[i]
				return nil
			}
		}
		return errors.New("人员不存在")
	}); err != nil {
		fail(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) deleteInstaller(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.st.Update(func(d *model.Data) error {
		for i := range d.Installers {
			if d.Installers[i].ID == id {
				d.Installers = append(d.Installers[:i], d.Installers[i+1:]...)
				// 同步移除部署任务中的分派记录
				for j := range d.Tasks {
					ids := d.Tasks[j].InstallerIDs[:0]
					for _, x := range d.Tasks[j].InstallerIDs {
						if x != id {
							ids = append(ids, x)
						}
					}
					d.Tasks[j].InstallerIDs = ids
				}
				// 验收单验收人置空
				for j := range d.Orders {
					if d.Orders[j].InspectorID == id {
						d.Orders[j].InspectorID = ""
					}
				}
				return nil
			}
		}
		return errors.New("人员不存在")
	}); err != nil {
		fail(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---- 部署任务 ----

func (a *API) listTasks(w http.ResponseWriter, _ *http.Request) {
	type resp struct {
		Stages []string               `json:"stages"`
		Tasks  []model.DeploymentTask `json:"tasks"`
	}
	var out resp
	out.Stages = model.DeploymentStages
	a.st.View(func(d *model.Data) { out.Tasks = d.Tasks })
	writeJSON(w, http.StatusOK, out)
}

func (a *API) createTask(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CategoryID string `json:"categoryId"`
		Project    string `json:"project"`
		Note       string `json:"note"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	var created *model.DeploymentTask
	if err := a.st.Update(func(d *model.Data) error {
		for _, c := range d.Categories {
			if c.ID != in.CategoryID {
				continue
			}
			project := in.Project
			if project == "" {
				project = "AI边缘算力中心一期建设"
			}
			t := model.DeploymentTask{
				ID: d.NextID("TASK", 100), Project: project,
				CategoryID: c.ID, CategoryName: c.Name, Quantity: c.Quantity, Unit: c.Unit,
				Stage: 0, Status: model.TaskPending, StartAt: time.Now().Format(time.RFC3339), Note: in.Note,
			}
			d.Tasks = append(d.Tasks, t)
			created = &t
			return nil
		}
		return errors.New("节点类别不存在")
	}); err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *API) setTaskStage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		Stage int `json:"stage"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	last := len(model.DeploymentStages) - 1
	if in.Stage < 0 || in.Stage > last {
		fail(w, http.StatusBadRequest, fmt.Sprintf("stage 取值 0~%d", last))
		return
	}
	var out *model.DeploymentTask
	if err := a.st.Update(func(d *model.Data) error {
		for i := range d.Tasks {
			if d.Tasks[i].ID != id {
				continue
			}
			d.Tasks[i].Stage = in.Stage
			switch {
			case in.Stage == 0:
				d.Tasks[i].Status = model.TaskPending
			case in.Stage == last:
				d.Tasks[i].Status = model.TaskDone
			default:
				d.Tasks[i].Status = model.TaskRunning
			}
			out = &d.Tasks[i]
			return nil
		}
		return errors.New("任务不存在")
	}); err != nil {
		fail(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) assignTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		InstallerIDs []string `json:"installerIds"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	var out *model.DeploymentTask
	if err := a.st.Update(func(d *model.Data) error {
		known := map[string]bool{}
		for _, ins := range d.Installers {
			known[ins.ID] = true
		}
		for _, x := range in.InstallerIDs {
			if !known[x] {
				return errors.New("安装人员不存在: " + x)
			}
		}
		for i := range d.Tasks {
			if d.Tasks[i].ID == id {
				d.Tasks[i].InstallerIDs = in.InstallerIDs
				out = &d.Tasks[i]
				return nil
			}
		}
		return errors.New("任务不存在")
	}); err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ---- 设备验收 ----

func (a *API) listOrders(w http.ResponseWriter, r *http.Request) {
	status, cat := r.URL.Query().Get("status"), r.URL.Query().Get("category")
	var out []model.AcceptanceOrder
	a.st.View(func(d *model.Data) {
		for _, o := range d.Orders {
			if status != "" && o.Status != status {
				continue
			}
			if cat != "" && o.CategoryID != cat {
				continue
			}
			out = append(out, o)
		}
	})
	if out == nil {
		out = []model.AcceptanceOrder{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var out *model.AcceptanceOrder
	a.st.View(func(d *model.Data) {
		for i := range d.Orders {
			if d.Orders[i].ID == id {
				c := d.Orders[i]
				out = &c
			}
		}
	})
	if out == nil {
		fail(w, http.StatusNotFound, "验收单不存在")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// createOrderFromTask 为部署任务生成下一台（组）设备的验收单。
func (a *API) createOrderFromTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		InspectorID string `json:"inspectorId"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	var created *model.AcceptanceOrder
	err := a.st.Update(func(d *model.Data) error {
		var task *model.DeploymentTask
		for i := range d.Tasks {
			if d.Tasks[i].ID == id {
				task = &d.Tasks[i]
				break
			}
		}
		if task == nil {
			return errors.New("部署任务不存在")
		}
		if task.Accepted >= task.Quantity {
			return fmt.Errorf("该任务共 %d %s，验收单已全部生成", task.Quantity, task.Unit)
		}
		if in.InspectorID != "" {
			ok := false
			for _, ins := range d.Installers {
				if ins.ID == in.InspectorID {
					ok = true
					break
				}
			}
			if !ok {
				return errors.New("验收人不存在")
			}
		}
		var cat *model.NodeCategory
		for i := range d.Categories {
			if d.Categories[i].ID == task.CategoryID {
				cat = &d.Categories[i]
				break
			}
		}
		if cat == nil {
			return errors.New("节点类别缺失")
		}
		now := time.Now().Format(time.RFC3339)
		o := model.AcceptanceOrder{
			ID: d.NextID("ACC", 1000), TaskID: task.ID,
			CategoryID: cat.ID, CategoryName: cat.Name,
			DeviceLabel: fmt.Sprintf("%s #%d", cat.Name, task.Accepted+1),
			InspectorID: in.InspectorID, Status: model.OrderPending,
			CreatedAt: now, UpdatedAt: now,
		}
		for _, t := range cat.Items {
			o.Items = append(o.Items, model.AcceptanceItem{
				Key: t.Key, Title: t.Title, Content: t.Content, Category: t.Category,
				Mandatory: t.Mandatory, Status: model.ItemPending,
			})
		}
		task.Accepted++
		d.Orders = append(d.Orders, o)
		created = &o
		return nil
	})
	if err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *API) updateOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		InspectorID *string `json:"inspectorId"`
		Conclusion  *string `json:"conclusion"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	var out *model.AcceptanceOrder
	if err := a.st.Update(func(d *model.Data) error {
		for i := range d.Orders {
			if d.Orders[i].ID != id {
				continue
			}
			if in.InspectorID != nil {
				d.Orders[i].InspectorID = *in.InspectorID
			}
			if in.Conclusion != nil {
				d.Orders[i].Conclusion = *in.Conclusion
			}
			d.Orders[i].UpdatedAt = time.Now().Format(time.RFC3339)
			c := d.Orders[i]
			out = &c
			return nil
		}
		return errors.New("验收单不存在")
	}); err != nil {
		fail(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// updateItem 更新验收条目结论并重算验收单状态。
func (a *API) updateItem(w http.ResponseWriter, r *http.Request) {
	id, key := r.PathValue("id"), r.PathValue("key")
	var in struct {
		Status    string `json:"status"`
		Result    string `json:"result"`
		Evidence  string `json:"evidence"`
		CheckedBy string `json:"checkedBy"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if !validItemStatus[in.Status] {
		fail(w, http.StatusBadRequest, "条目状态必须为：待检/进行中/合格/不合格/不适用")
		return
	}
	if in.Status == model.ItemPass && strings.TrimSpace(in.Result) == "" {
		fail(w, http.StatusBadRequest, "判定合格时必须填写验收结果说明")
		return
	}
	var out *model.AcceptanceOrder
	if err := a.st.Update(func(d *model.Data) error {
		for i := range d.Orders {
			if d.Orders[i].ID != id {
				continue
			}
			for j := range d.Orders[i].Items {
				if d.Orders[i].Items[j].Key != key {
					continue
				}
				it := &d.Orders[i].Items[j]
				it.Status = in.Status
				it.Result = in.Result
				it.Evidence = in.Evidence
				it.CheckedBy = in.CheckedBy
				it.CheckedAt = time.Now().Format(time.RFC3339)
				d.Orders[i].Status = model.RecalcOrderStatus(d.Orders[i].Items)
				d.Orders[i].UpdatedAt = it.CheckedAt
				if d.Orders[i].Status == model.OrderPassed {
					d.Orders[i].FinishedAt = it.CheckedAt
				} else {
					d.Orders[i].FinishedAt = ""
				}
				c := d.Orders[i]
				out = &c
				return nil
			}
			return errors.New("验收条目不存在: " + key)
		}
		return errors.New("验收单不存在")
	}); err != nil {
		fail(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// applyStress 引用压测结果自动回填验收条目。
// type: http（并发压测）| burnin（满载烤机）
func (a *API) applyStress(w http.ResponseWriter, r *http.Request) {
	id, key := r.PathValue("id"), r.PathValue("key")
	var in struct {
		StressID string `json:"stressId"`
		Type     string `json:"type"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if in.Type != "http" && in.Type != "burnin" {
		fail(w, http.StatusBadRequest, "type 必须为 http 或 burnin")
		return
	}

	var out *model.AcceptanceOrder
	if err := a.st.Update(func(d *model.Data) error {
		// 1) 查找压测结果并生成结论
		var verdict, result string
		if in.Type == "http" {
			for _, t := range d.HTTPTests {
				if t.ID != in.StressID || t.Result == nil {
					continue
				}
				passTxt := map[bool]string{true: "合格", false: "不合格"}[t.Result.Pass]
				verdict = map[bool]string{true: model.ItemPass, false: model.ItemFail}[t.Result.Pass]
				result = fmt.Sprintf("并发压测 %s：并发%d、持续%ds、RPS %.1f、P95 %.0fms（阈值%.0fms）、错误率%.2f%%（阈值%.2f%%）→ %s",
					t.ID, t.Concurrency, t.Result.DurationSec, t.Result.RPS, t.Result.P95,
					t.MaxP95Ms, t.Result.ErrRatePct, t.MaxErrRatePct, passTxt)
				break
			}
		} else {
			for _, t := range d.BurnIns {
				if t.ID != in.StressID || t.Result == nil {
					continue
				}
				verdict = map[bool]string{true: model.ItemPass, false: model.ItemFail}[t.Result.Pass]
				result = fmt.Sprintf("满载烤机 %s：%s", t.ID, t.Result.Summary)
				break
			}
		}
		if verdict == "" {
			return errors.New("压测记录不存在或尚未完成")
		}

		// 2) 回填验收条目并重算验收单状态
		for i := range d.Orders {
			if d.Orders[i].ID != id {
				continue
			}
			for j := range d.Orders[i].Items {
				if d.Orders[i].Items[j].Key != key {
					continue
				}
				it := &d.Orders[i].Items[j]
				it.Status = verdict
				it.Result = result
				it.Evidence = "压测记录 " + in.StressID
				if d.Orders[i].InspectorID != "" {
					it.CheckedBy = d.Orders[i].InspectorID
				}
				it.CheckedAt = time.Now().Format(time.RFC3339)
				d.Orders[i].Status = model.RecalcOrderStatus(d.Orders[i].Items)
				d.Orders[i].UpdatedAt = it.CheckedAt
				if d.Orders[i].Status == model.OrderPassed {
					d.Orders[i].FinishedAt = it.CheckedAt
				}
				c := d.Orders[i]
				out = &c
				return nil
			}
			return errors.New("验收条目不存在: " + key)
		}
		return errors.New("验收单不存在")
	}); err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
