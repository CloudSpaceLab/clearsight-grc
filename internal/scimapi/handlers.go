package scimapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/elimity-com/scim"
	scimerrors "github.com/elimity-com/scim/errors"
	scimvalidator "github.com/elimity-com/scim/filter"
	scimfilter "github.com/scim2/filter-parser/v2"
)

type userHandler struct{ repository Repository }
type groupHandler struct{ repository Repository }

func (h *userHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	source, err := sourceForRequest(r)
	if err != nil {
		return scim.Resource{}, err
	}
	user, err := h.repository.CreateUser(r.Context(), source, userFromAttributes(attributes))
	return resourceFromUser(user), mapRepositoryError(err, "user")
}

func (h *userHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	source, err := sourceForRequest(r)
	if err != nil {
		return scim.Resource{}, err
	}
	user, err := h.repository.GetUser(r.Context(), source, id)
	return resourceFromUser(user), mapRepositoryError(err, id)
}

func (h *userHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	source, err := sourceForRequest(r)
	if err != nil {
		return scim.Page{}, err
	}
	filter, err := userFilter(params.FilterValidator)
	if err != nil {
		return scim.Page{}, err
	}
	users, total, err := h.repository.ListUsers(r.Context(), source, filter, max(params.StartIndex-1, 0), params.Count)
	if err != nil {
		return scim.Page{}, mapRepositoryError(err, "users")
	}
	resources := make([]scim.Resource, 0, len(users))
	for _, user := range users {
		resources = append(resources, resourceFromUser(user))
	}
	return scim.Page{TotalResults: total, Resources: resources}, nil
}

func (h *userHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	source, err := sourceForRequest(r)
	if err != nil {
		return scim.Resource{}, err
	}
	user, err := h.repository.ReplaceUser(r.Context(), source, id, userFromAttributes(attributes))
	return resourceFromUser(user), mapRepositoryError(err, id)
}

func (h *userHandler) Delete(r *http.Request, id string) error {
	source, err := sourceForRequest(r)
	if err != nil {
		return err
	}
	return mapRepositoryError(h.repository.DeleteUser(r.Context(), source, id), id)
}

func (h *userHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	source, err := sourceForRequest(r)
	if err != nil {
		return scim.Resource{}, err
	}
	current, err := h.repository.GetUser(r.Context(), source, id)
	if err != nil {
		return scim.Resource{}, mapRepositoryError(err, id)
	}
	if err := applyUserPatch(&current, operations); err != nil {
		return scim.Resource{}, err
	}
	updated, err := h.repository.ReplaceUser(r.Context(), source, id, current)
	return resourceFromUser(updated), mapRepositoryError(err, id)
}

func (h *groupHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	source, err := sourceForRequest(r)
	if err != nil {
		return scim.Resource{}, err
	}
	group, err := groupFromAttributes(attributes)
	if err != nil {
		return scim.Resource{}, err
	}
	group, err = h.repository.CreateGroup(r.Context(), source, group)
	return resourceFromGroup(group), mapRepositoryError(err, "group")
}

func (h *groupHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	source, err := sourceForRequest(r)
	if err != nil {
		return scim.Resource{}, err
	}
	group, err := h.repository.GetGroup(r.Context(), source, id)
	return resourceFromGroup(group), mapRepositoryError(err, id)
}

func (h *groupHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	source, err := sourceForRequest(r)
	if err != nil {
		return scim.Page{}, err
	}
	filter, err := groupFilter(params.FilterValidator)
	if err != nil {
		return scim.Page{}, err
	}
	groups, total, err := h.repository.ListGroups(r.Context(), source, filter, max(params.StartIndex-1, 0), params.Count)
	if err != nil {
		return scim.Page{}, mapRepositoryError(err, "groups")
	}
	resources := make([]scim.Resource, 0, len(groups))
	for _, group := range groups {
		resources = append(resources, resourceFromGroup(group))
	}
	return scim.Page{TotalResults: total, Resources: resources}, nil
}

func (h *groupHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	source, err := sourceForRequest(r)
	if err != nil {
		return scim.Resource{}, err
	}
	group, err := groupFromAttributes(attributes)
	if err != nil {
		return scim.Resource{}, err
	}
	group, err = h.repository.ReplaceGroup(r.Context(), source, id, group)
	return resourceFromGroup(group), mapRepositoryError(err, id)
}

func (h *groupHandler) Delete(r *http.Request, id string) error {
	source, err := sourceForRequest(r)
	if err != nil {
		return err
	}
	return mapRepositoryError(h.repository.DeleteGroup(r.Context(), source, id), id)
}

func (h *groupHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	source, err := sourceForRequest(r)
	if err != nil {
		return scim.Resource{}, err
	}
	current, err := h.repository.GetGroup(r.Context(), source, id)
	if err != nil {
		return scim.Resource{}, mapRepositoryError(err, id)
	}
	if err := applyGroupPatch(&current, operations); err != nil {
		return scim.Resource{}, err
	}
	updated, err := h.repository.ReplaceGroup(r.Context(), source, id, current)
	return resourceFromGroup(updated), mapRepositoryError(err, id)
}

func userFromAttributes(attributes scim.ResourceAttributes) User {
	return User{
		ExternalID: externalID(attributes),
		UserName:   stringAttribute(attributes, "userName"),
		DisplayName: func() string {
			value := stringAttribute(attributes, "displayName")
			if value == "" {
				return stringAttribute(attributes, "userName")
			}
			return value
		}(),
		Active: boolAttribute(attributes, "active", true),
	}
}

func groupFromAttributes(attributes scim.ResourceAttributes) (Group, error) {
	group := Group{ExternalID: externalID(attributes), DisplayName: stringAttribute(attributes, "displayName")}
	members, err := parseMembers(attributes["members"])
	if err != nil {
		return Group{}, err
	}
	group.Members = members
	return group, nil
}

func parseMembers(value any) ([]GroupMember, error) {
	if value == nil {
		return []GroupMember{}, nil
	}
	values, ok := value.([]interface{})
	if !ok {
		return nil, scimerrors.ScimErrorBadRequest("members must be an array")
	}
	result := make([]GroupMember, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		item, ok := raw.(map[string]interface{})
		if !ok {
			return nil, scimerrors.ScimErrorBadRequest("group member must be an object")
		}
		memberType, _ := item["type"].(string)
		if memberType != "" && !strings.EqualFold(memberType, "User") {
			return nil, scimerrors.ScimErrorBadRequest("nested groups are not supported; group members must be Users")
		}
		id, _ := item["value"].(string)
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, scimerrors.ScimErrorBadRequest("group member value is required")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		display, _ := item["display"].(string)
		result = append(result, GroupMember{UserID: id, DisplayName: strings.TrimSpace(display)})
	}
	return result, nil
}

func userFilter(validator *scimvalidator.Validator) (UserFilter, error) {
	if validator == nil {
		return UserFilter{}, nil
	}
	attribute, ok := validator.GetFilter().(*scimfilter.AttributeExpression)
	if !ok || attribute.Operator != scimfilter.EQ {
		return UserFilter{}, scimerrors.ScimErrorBadRequest("only equality filters on userName or externalId are supported")
	}
	value, ok := attribute.CompareValue.(string)
	if !ok {
		return UserFilter{}, scimerrors.ScimErrorBadRequest("filter value must be a string")
	}
	switch strings.ToLower(attribute.AttributePath.AttributeName) {
	case "username":
		return UserFilter{UserName: value}, nil
	case "externalid":
		return UserFilter{ExternalID: value}, nil
	default:
		return UserFilter{}, scimerrors.ScimErrorBadRequest("only userName and externalId filters are supported")
	}
}

func groupFilter(validator *scimvalidator.Validator) (GroupFilter, error) {
	if validator == nil {
		return GroupFilter{}, nil
	}
	attribute, ok := validator.GetFilter().(*scimfilter.AttributeExpression)
	if !ok || attribute.Operator != scimfilter.EQ {
		return GroupFilter{}, scimerrors.ScimErrorBadRequest("only equality filters on displayName or externalId are supported")
	}
	value, ok := attribute.CompareValue.(string)
	if !ok {
		return GroupFilter{}, scimerrors.ScimErrorBadRequest("filter value must be a string")
	}
	switch strings.ToLower(attribute.AttributePath.AttributeName) {
	case "displayname":
		return GroupFilter{DisplayName: value}, nil
	case "externalid":
		return GroupFilter{ExternalID: value}, nil
	default:
		return GroupFilter{}, scimerrors.ScimErrorBadRequest("only displayName and externalId filters are supported")
	}
}

func applyUserPatch(user *User, operations []scim.PatchOperation) error {
	for _, operation := range operations {
		path := ""
		if operation.Path != nil {
			path = strings.ToLower(operation.Path.AttributePath.AttributeName)
		}
		if path == "" {
			values, ok := operation.Value.(map[string]interface{})
			if !ok {
				return scimerrors.ScimErrorBadRequest("path-less user PATCH value must be an object")
			}
			if value, ok := values["userName"].(string); ok {
				user.UserName = strings.TrimSpace(value)
			}
			if value, ok := values["displayName"].(string); ok {
				user.DisplayName = strings.TrimSpace(value)
			}
			if value, ok := values["externalId"].(string); ok {
				user.ExternalID = strings.TrimSpace(value)
			}
			if value, ok := values["active"].(bool); ok {
				user.Active = value
			}
			continue
		}
		switch path {
		case "username", "displayname", "externalid":
			if operation.Op == scim.PatchOperationRemove {
				if path == "username" {
					return scimerrors.ScimErrorBadRequest("userName cannot be removed")
				}
				if path == "displayname" {
					user.DisplayName = user.UserName
				} else {
					user.ExternalID = ""
				}
				continue
			}
			value, ok := operation.Value.(string)
			if !ok {
				return scimerrors.ScimErrorBadRequest(path + " must be a string")
			}
			value = strings.TrimSpace(value)
			switch path {
			case "username":
				user.UserName = value
			case "displayname":
				user.DisplayName = value
			case "externalid":
				user.ExternalID = value
			}
		case "active":
			if operation.Op == scim.PatchOperationRemove {
				return scimerrors.ScimErrorBadRequest("active cannot be removed")
			}
			value, ok := operation.Value.(bool)
			if !ok {
				return scimerrors.ScimErrorBadRequest("active must be a boolean")
			}
			user.Active = value
		default:
			return scimerrors.ScimErrorBadRequest("unsupported user PATCH path: " + path)
		}
	}
	return nil
}

func applyGroupPatch(group *Group, operations []scim.PatchOperation) error {
	for _, operation := range operations {
		path := ""
		if operation.Path != nil {
			path = strings.ToLower(operation.Path.AttributePath.AttributeName)
		}
		if path == "" {
			values, ok := operation.Value.(map[string]interface{})
			if !ok {
				return scimerrors.ScimErrorBadRequest("path-less group PATCH value must be an object")
			}
			if value, ok := values["displayName"].(string); ok {
				group.DisplayName = strings.TrimSpace(value)
			}
			if value, ok := values["externalId"].(string); ok {
				group.ExternalID = strings.TrimSpace(value)
			}
			if value, exists := values["members"]; exists {
				members, err := parseMembers(value)
				if err != nil {
					return err
				}
				group.Members = members
			}
			continue
		}

		switch path {
		case "displayname", "externalid":
			if operation.Op == scim.PatchOperationRemove {
				if path == "displayname" {
					return scimerrors.ScimErrorBadRequest("displayName cannot be removed")
				}
				group.ExternalID = ""
				continue
			}
			value, ok := operation.Value.(string)
			if !ok {
				return scimerrors.ScimErrorBadRequest(path + " must be a string")
			}
			if path == "displayname" {
				group.DisplayName = strings.TrimSpace(value)
			} else {
				group.ExternalID = strings.TrimSpace(value)
			}
		case "members":
			switch operation.Op {
			case scim.PatchOperationAdd:
				members, err := parseMembers(operation.Value)
				if err != nil {
					return err
				}
				group.Members = mergeMembers(group.Members, members)
			case scim.PatchOperationReplace:
				members, err := parseMembers(operation.Value)
				if err != nil {
					return err
				}
				group.Members = members
			case scim.PatchOperationRemove:
				if operation.Path.ValueExpression == nil {
					group.Members = []GroupMember{}
					continue
				}
				attribute, ok := operation.Path.ValueExpression.(*scimfilter.AttributeExpression)
				if !ok || attribute.Operator != scimfilter.EQ || !strings.EqualFold(attribute.AttributePath.AttributeName, "value") {
					return scimerrors.ScimErrorBadRequest("member removal must filter by value eq resource-id")
				}
				id, ok := attribute.CompareValue.(string)
				if !ok {
					return scimerrors.ScimErrorBadRequest("member removal value must be a string")
				}
				group.Members = removeMember(group.Members, strings.TrimSpace(id))
			default:
				return scimerrors.ScimErrorBadRequest("unsupported members PATCH operation")
			}
		default:
			return scimerrors.ScimErrorBadRequest("unsupported group PATCH path: " + path)
		}
	}
	return nil
}

func mergeMembers(current, additions []GroupMember) []GroupMember {
	result := append([]GroupMember(nil), current...)
	seen := make(map[string]struct{}, len(result))
	for _, member := range result {
		seen[member.UserID] = struct{}{}
	}
	for _, member := range additions {
		if _, exists := seen[member.UserID]; exists {
			continue
		}
		seen[member.UserID] = struct{}{}
		result = append(result, member)
	}
	return result
}

func removeMember(current []GroupMember, id string) []GroupMember {
	result := current[:0]
	for _, member := range current {
		if member.UserID != id {
			result = append(result, member)
		}
	}
	return result
}

func mapRepositoryError(err error, id string) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return scimerrors.ScimErrorResourceNotFound(id)
	case errors.Is(err, ErrConflict):
		return scimerrors.ScimErrorUniqueness
	case errors.Is(err, ErrInvalid):
		return scimerrors.ScimErrorBadRequest(err.Error())
	default:
		return fmt.Errorf("SCIM repository operation failed: %w", err)
	}
}
