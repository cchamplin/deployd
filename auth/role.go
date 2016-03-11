package auth

type Role struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	Permissions Permissions `json:"permissions"`
}

type Roles [256]Role

func initBuiltin(roles Roles) {
	var role Role
	role = Role{}
	role.Id = "anonymous"
	role.Name = "Anonymous Users"
	var permissions Permissions
	permissions = make([]Permission, 1)
	permissions[0] = Permission{Name: "Login", Flags: READ & CREATE & UPDATE}
	role.Permissions = permissions
	roles[0] = role

	role = Role{}
	role.Id = "administrator"
	role.Name = "Administrator"
	permissions = make([]Permission, 11)
	permissions[0] = Permission{Name: "Login", Flags: READ & CREATE & UPDATE & DELETE}
	permissions[1] = Permission{Name: "Index", Flags: READ}
	permissions[2] = Permission{Name: "Packages", Flags: READ & CREATE}
	permissions[3] = Permission{Name: "PackageDetails", Flags: READ & UPDATE & DELETE}
	permissions[4] = Permission{Name: "PackageDeploy", Flags: CREATE}
	permissions[5] = Permission{Name: "PackageDeployTemplate", Flags: CREATE}
	permissions[6] = Permission{Name: "Deployments", Flags: READ}
	permissions[7] = Permission{Name: "DeploymentDetails", Flags: READ}
	permissions[8] = Permission{Name: "CurrentUser", Flags: READ}
	permissions[9] = Permission{Name: "Users", Flags: READ & CREATE}
	permissions[10] = Permission{Name: "UserDetails", Flags: READ & CREATE & DELETE}
	role.Permissions = permissions
	roles[1] = role

}
