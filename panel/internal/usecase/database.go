package usecase

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/repository"
)

type DatabaseUseCase struct {
	repo       repository.DatabaseRepo
	mysqlDSN   string // root DSN to node MySQL e.g. root:pass@tcp(node:3306)/
	mysqlHost  string
	mysqlPort  int
	encryptKey []byte // 32 bytes AES-256
}

func NewDatabaseUseCase(repo repository.DatabaseRepo, mysqlDSN, mysqlHost string, mysqlPort int, encryptKey []byte) *DatabaseUseCase {
	return &DatabaseUseCase{repo: repo, mysqlDSN: mysqlDSN, mysqlHost: mysqlHost, mysqlPort: mysqlPort, encryptKey: encryptKey}
}

func (uc *DatabaseUseCase) Create(ctx context.Context, serverID string) (*domain.ServerDatabase, string, error) {
	shortID := serverID
	if len(shortID) > 8 { shortID = shortID[:8] }
	dbName := fmt.Sprintf("s%s_db%d", shortID, time.Now().Unix()%10000)
	dbUser := fmt.Sprintf("u%s", shortID)
	dbPass, err := generatePassword(24)
	if err != nil { return nil, "", err }

	db, err := sql.Open("mysql", uc.mysqlDSN)
	if err != nil { return nil, "", fmt.Errorf("mysql connect: %w", err) }
	defer db.Close()

	queries := []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName),
		fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", dbUser, dbPass),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'", dbName, dbUser),
		"FLUSH PRIVILEGES",
	}
	for _, q := range queries {
		if _, err := db.ExecContext(ctx, q); err != nil {
			return nil, "", fmt.Errorf("mysql exec %q: %w", q, err)
		}
	}

	enc, err := uc.encrypt(dbPass)
	if err != nil { return nil, "", err }

	d := &domain.ServerDatabase{
		ID:        uuid.NewString(),
		ServerID:  serverID,
		DBName:    dbName,
		DBUser:    dbUser,
		DBPassEnc: enc,
		Host:      uc.mysqlHost,
		Port:      uc.mysqlPort,
		CreatedAt: time.Now(),
	}
	if err := uc.repo.Create(ctx, d); err != nil {
		return nil, "", fmt.Errorf("save db record: %w", err)
	}
	return d, dbPass, nil
}

func (uc *DatabaseUseCase) List(ctx context.Context, serverID string) ([]*domain.ServerDatabase, error) {
	return uc.repo.ListByServer(ctx, serverID)
}

func (uc *DatabaseUseCase) Delete(ctx context.Context, id string) error {
	d, err := uc.repo.GetByID(ctx, id)
	if err != nil { return fmt.Errorf("db record not found: %w", err) }

	db, err := sql.Open("mysql", uc.mysqlDSN)
	if err != nil { return fmt.Errorf("mysql connect: %w", err) }
	defer db.Close()

	ctx2 := context.Background()
	_, _ = db.ExecContext(ctx2, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", d.DBName))
	_, _ = db.ExecContext(ctx2, fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", d.DBUser))
	_, _ = db.ExecContext(ctx2, "FLUSH PRIVILEGES")

	return uc.repo.Delete(ctx, id)
}

func (uc *DatabaseUseCase) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(uc.encryptKey)
	if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block)
	if err != nil { return "", err }
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil { return "", err }
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func generatePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil { return "", err }
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}
