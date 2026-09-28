package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"project.local/commerce-api/internal/admin"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
)

const recoveryConfirmation = "RECOVER_LAST_SUPER_ADMIN"

type recoveryInput struct {
	Identifier string
	Operator   string
	Password   string
	ResetMFA   bool
}

func main() {
	log.SetFlags(0)

	input, err := loadInput()
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.NewPostgres(ctx, cfg)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	result, err := admin.RecoverLastSuperAdmin(
		ctx,
		db,
		admin.SuperAdminBreakGlassInput{
			Identifier:  input.Identifier,
			NewPassword: input.Password,
			ResetMFA:    input.ResetMFA,
			Operator:    input.Operator,
		},
	)
	if err != nil {
		log.Fatalf("Super Admin break-glass recovery failed: %v", err)
	}

	fmt.Println("Super Admin break-glass recovery complete.")
	fmt.Printf("Staff ID: %s\n", result.StaffAccountID)
	fmt.Printf("Staff code: %s\n", result.StaffCode)
	fmt.Printf("Email: %s\n", result.Email)
	fmt.Println("Password reset: yes")
	if result.MFAReset {
		fmt.Println("MFA reset: yes; new MFA enrollment will be required on the next login")
	} else {
		fmt.Println("MFA reset: no; existing MFA remains required")
	}
	fmt.Println("Existing staff/Admin sessions and pending Admin login challenges were revoked.")
	fmt.Println("Password was not printed.")
}

func loadInput() (recoveryInput, error) {
	var input recoveryInput

	flag.StringVar(
		&input.Identifier,
		"identifier",
		"",
		"exact Super Admin staff code, email, or staff UUID",
	)
	flag.StringVar(
		&input.Operator,
		"operator",
		"",
		"human/operator identity recorded in the security audit event",
	)
	flag.BoolVar(
		&input.ResetMFA,
		"reset-mfa",
		false,
		"disable existing TOTP and invalidate MFA recovery codes",
	)
	flag.Parse()

	input.Identifier = strings.TrimSpace(input.Identifier)
	input.Operator = strings.TrimSpace(input.Operator)
	input.Password = os.Getenv("ADMIN_RECOVERY_PASSWORD")

	if input.Identifier == "" {
		return recoveryInput{}, fmt.Errorf("--identifier is required")
	}
	if input.Operator == "" {
		return recoveryInput{}, fmt.Errorf("--operator is required")
	}
	if strings.TrimSpace(os.Getenv("ADMIN_RECOVERY_CONFIRM")) != recoveryConfirmation {
		return recoveryInput{}, fmt.Errorf(
			"ADMIN_RECOVERY_CONFIRM must equal %q",
			recoveryConfirmation,
		)
	}
	if input.Password == "" {
		return recoveryInput{}, fmt.Errorf("ADMIN_RECOVERY_PASSWORD is required")
	}

	return input, nil
}
