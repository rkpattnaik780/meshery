package model

import (
	"strings"

	"github.com/layer5io/meshery/models"
	"github.com/layer5io/meshkit/utils"
	meshsyncmodel "github.com/layer5io/meshsync/pkg/model"

	corev1 "k8s.io/api/core/v1"
)

func GetControlPlaneState(selectors []MeshType, provider models.Provider, cid string) ([]*ControlPlane, error) {
	object := []meshsyncmodel.Object{}
	controlplanelist := make([]*ControlPlane, 0)

	for _, selector := range selectors {
		result := provider.GetGenericPersister().Model(&meshsyncmodel.Object{}).
			Where("cluster_id = ?", cid).
			Preload("ObjectMeta", "namespace = ?", controlPlaneNamespace[MeshType(selector)]).
			Preload("ObjectMeta.Labels", "kind = ?", meshsyncmodel.KindLabel).
			Preload("ObjectMeta.Annotations", "kind = ?", meshsyncmodel.KindAnnotation).
			Preload("Spec").
			Preload("Status").
			Find(&object, "kind = ?", "Pod")
		if result.Error != nil {
			return nil, ErrQuery(result.Error)
		}
		members := make([]*ControlPlaneMember, 0)
		for _, obj := range object {
			if meshsyncmodel.IsObject(obj) {
				objspec := corev1.PodSpec{}
				err := utils.Unmarshal(obj.Spec.Attribute, &objspec)
				if err != nil {
					return nil, err
				}
				var imageOrgs = make(map[string]bool)
				for _, c := range objspec.Containers {
					imageOrgs[strings.Split(c.Image, "/")[1]] = true // Extracting image org from <domainname>/<imageorg>/<imagename>
				}
				version := "unknown"

				//If image orgs are not passed on in from controlPlaneImageOrgs variable, then skip this filtering (for backward compatibility)
				if len(controlPlaneImageOrgs[MeshType(selector)]) != 0 && !haveCommonElements(controlPlaneImageOrgs[MeshType(selector)], imageOrgs) {
					continue
				}
				if len(strings.Split(objspec.Containers[0].Image, ":")) > 0 {
					version = strings.Split(objspec.Containers[0].Image, ":")[1]
				}
				members = append(members, &ControlPlaneMember{
					Name:      obj.ObjectMeta.Name,
					Component: strings.Split(obj.ObjectMeta.GenerateName, "-")[0],
					Version:   strings.Split(version, "@")[0],
					Namespace: obj.ObjectMeta.Namespace,
				})
			}
		}
		controlplanelist = append(controlplanelist, &ControlPlane{
			Name:    strings.ToLower(selector.String()),
			Members: members,
		})
	}

	return controlplanelist, nil
}
func haveCommonElements(a []string, b map[string]bool) bool {
	for _, ae := range a {
		if b[ae] {
			return true
		}
	}
	return false
}
