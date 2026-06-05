package service

import "testing"

func TestParseHomeExtIdentityByUserCategory(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		role     string
		roleName string
		category string
	}{
		{
			name:     "student",
			html:     `<form action="/eams/homeExt.action"><span>学生</span><input type="hidden" name="security.userCategoryId" value="1"/></form>`,
			role:     "student",
			roleName: "学生",
			category: "1",
		},
		{
			name:     "teacher",
			html:     `<form action="/eams/homeExt.action"><span>教师</span><input type="hidden" name="security.userCategoryId" value="2"/></form>`,
			role:     "teacher",
			roleName: "教师",
			category: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := ParseHomeExtIdentity(tt.html)
			if identity.Role != tt.role || identity.RoleName != tt.roleName || identity.UserCategoryID != tt.category {
				t.Fatalf("unexpected identity: %+v", identity)
			}
		})
	}
}

func TestParseHomeExtIdentityFallbackMarkers(t *testing.T) {
	student := ParseHomeExtIdentity(`<a href="/eams/courseTableForStd.action">我的课表</a>`)
	if student.Role != "student" {
		t.Fatalf("expected student fallback, got %+v", student)
	}

	teacher := ParseHomeExtIdentity(`<a href="/eams/courseTableForTeacher.action">我的课程</a>`)
	if teacher.Role != "teacher" {
		t.Fatalf("expected teacher fallback, got %+v", teacher)
	}
}
