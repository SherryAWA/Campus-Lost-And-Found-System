package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
)

func corsMiddleware(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.Next()
}

var (
	logger *logrus.Logger
	DB     *sql.DB
)

func initDB() {
	var err error
	dbUser := "root"
	dbPassword := "040801"
	dbHost := "localhost"
	dbName := "campus"
	dbConnectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbName)
	DB, err = sql.Open("mysql", dbConnectionString)
	if err != nil {
		logger.Fatalf("Error connecting to the database: %v", err)
	}
	if err = DB.Ping(); err != nil {
		logger.Fatalf("Error pinging the database: %v", err)
	}
}

func init() {
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	initDB()
}

type Response struct {
	Success int         `json:"success"`
	Message string      `json:"message"`
	Token   string      `json:"token"`
	Data    interface{} `json:"data"`
}

func main() {
	gin.SetMode(gin.DebugMode)
	router := gin.Default()
	router.Use(corsMiddleware)
	router.POST("/login", loginHandler)
	router.POST("/register", registerHandler)
	router.POST("/userinfo", userInfoHandler)
	router.POST("/admininfo", adminInfoHandler)
	router.POST("/lossitem", lossItemHandler)
	router.POST("/founditem", foundItemHandler)
	router.POST("/addlossitem", addLossItemHandler)
	router.POST("/confirmclaim", confirmClaimHandler)
	router.POST("/addfounditem", addFoundItemHandler)
	router.POST("/getclaimforms", getClaimFormsHandler)
	router.POST("/complaint", complainHandler)
	router.POST("/complaints", complaintHandler)
	router.POST("/reclaiminfo", ReclaimInfoHandler)
	router.POST("/processreclaim", ProcessReclaimHandler)
	router.POST("/complaintinfo", ComplaintInfoHandler)
	router.POST("/processcomplaint", ProcessComplaintHandler)

	router.Run(":8080")
}

func loginHandler(c *gin.Context) {
	var loginReq struct {
		ID       string `json:"id"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		logger.Error("Failed to bind JSON: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var userPassword string
	var userStatus int
	var err error

	// 先查询User表
	err = DB.QueryRow("SELECT Password, Status FROM User WHERE ID = ?", loginReq.ID).Scan(&userPassword, &userStatus)
	if err == nil {
		if userStatus == 1 {
			logger.Warn("User account is frozen: ", loginReq.ID)
			c.JSON(http.StatusOK, Response{Success: 0, Message: "账号违规多次,已被冻结"})
			return
		}
		if userPassword != loginReq.Password {
			logger.Warn("Invalid password attempt for user: ", loginReq.ID)
			c.JSON(http.StatusOK, Response{Success: 0, Message: "密码错误"})
			return
		}
		logger.Info("Login successful for user: ", loginReq.ID)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"id":  loginReq.ID,
			"exp": time.Now().Add(5 * time.Minute).Unix(),
		})
		mySigningKey := []byte("ushjlwmwnwht")
		usertoken, _ := token.SignedString(mySigningKey)
		c.JSON(http.StatusOK, Response{Success: 1, Message: "登录成功", Token: usertoken})
	} else if err == sql.ErrNoRows {
		// 如果User表中没有找到，再查询Admin表
		err = DB.QueryRow("SELECT Password FROM Admin WHERE ID = ?", loginReq.ID).Scan(&userPassword)
		if err == sql.ErrNoRows {
			logger.Warn("User or Admin does not exist: ", loginReq.ID)
			c.JSON(http.StatusOK, Response{Success: 0, Message: "用户不存在"})
			return
		}
		if err != nil {
			logger.Error("Error querying Admin table: ", err)
			c.JSON(http.StatusInternalServerError, Response{Success: 0, Message: "内部服务器错误"})
			return
		}
		if userPassword != loginReq.Password {
			logger.Warn("Invalid password attempt for admin: ", loginReq.ID)
			c.JSON(http.StatusOK, Response{Success: 0, Message: "密码错误"})
			return
		}
		logger.Info("Login successful for admin: ", loginReq.ID)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"id":  loginReq.ID,
			"exp": time.Now().Add(5 * time.Minute).Unix(),
		})
		mySigningKey := []byte("ushjlwmwnwht")
		usertoken, _ := token.SignedString(mySigningKey)
		c.JSON(http.StatusOK, Response{Success: 2, Message: "登录成功", Token: usertoken})
	} else {
		logger.Error("Error querying User table: ", err)
		c.JSON(http.StatusInternalServerError, Response{Success: 0, Message: "内部服务器错误"})
		return
	}
}
func registerHandler(c *gin.Context) {
	var registerReq struct {
		ID       string `json:"id"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&registerReq); err != nil {
		logger.Error("Failed to bind JSON: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userCount int
	err := DB.QueryRow("SELECT COUNT(*) FROM Student WHERE ID = ?", registerReq.ID).Scan(&userCount)
	// logger.Info("User count: ", userCount)
	if err != nil {
		logger.Error("Error querying User table: ", err)
		c.JSON(http.StatusInternalServerError, Response{Success: 0, Message: "内部服务器错误"})
		return
	}

	if userCount == 0 {
		logger.Warn("Not found: ", registerReq.ID)
		c.JSON(http.StatusOK, Response{Success: 0, Message: "学号不存在"})
		return
	}

	if userCount > 1 {
		logger.Warn("Already registered: ", registerReq.ID)
		c.JSON(http.StatusOK, Response{Success: 0, Message: "该用户名已被注册"})
		return
	}

	_, err = DB.Exec("INSERT INTO User (ID, Password, Status) VALUES (?, ?, 0)", registerReq.ID, registerReq.Password)
	if err != nil {
		logger.Error("Error inserting into User table: ", err)
		c.JSON(http.StatusInternalServerError, Response{Success: 0, Message: "内部服务器错误"})
		return
	}
	logger.Info("Registration successful for user: ", registerReq.ID)
	c.JSON(http.StatusOK, Response{Success: 1, Message: "注册成功"})
}
func userInfoHandler(c *gin.Context) {
	// 从请求体中获取Token
	var req struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("Failed to bind JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "请求体格式错误"})
		return
	}

	// 解析Token
	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte("ushjlwmwnwht"), nil
	})
	if err != nil || !token.Valid {
		logger.Errorf("invalid Token: %v", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的Token"})
		return
	}

	// 从Token的claims中获取用户ID
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.Error("Token解析错误")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Token解析错误"})
		return
	}
	userID := claims["id"].(string)

	// 连接User和Student表查询用户信息
	var userName string
	var userTel string
	err = DB.QueryRow("SELECT s.Name, s.Telephone FROM User u INNER JOIN Student s ON u.id = s.ID WHERE u.ID = ?", userID).Scan(&userName, &userTel)
	if err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		} else {
			logger.Errorf("数据库查询错误: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		}
		return
	}

	// 返回用户信息
	c.JSON(http.StatusOK, gin.H{"id": userID, "name": userName, "telephone": userTel})
}

func adminInfoHandler(c *gin.Context) {
	// 从请求体中获取Token
	var req struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("Failed to bind JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "请求体格式错误"})
		return
	}

	// 解析Token
	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte("ushjlwmwnwht"), nil
	})
	if err != nil || !token.Valid {
		logger.Errorf("invalid Token: %v", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的Token"})
		return
	}

	// 从Token的claims中获取用户ID
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.Error("Token解析错误")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Token解析错误"})
		return
	}
	userID := claims["id"].(string)

	// 查询管理员信息
	var userPassword string
	err = DB.QueryRow("SELECT ID, Password FROM Admin WHERE ID = ?", userID).Scan(&userID, &userPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "管理员不存在"})
		} else {
			logger.Errorf("数据库查询错误: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "获取管理员信息失败"})
		}
		return
	}

	// 返回管理员信息
	c.JSON(http.StatusOK, gin.H{"id": userID, "password": userPassword})
}

type LossItem struct {
	ItemID      string `json:"ItemID"`
	Category    string `json:"Category"`
	ItemName    string `json:"ItemName"`
	Description string `json:"Description"`
	Location    string `json:"Location"`
	Time        string `json:"Time"`
}
type foundItem struct {
	ItemID      string `json:"ItemID"`
	Category    string `json:"Category"`
	ItemName    string `json:"ItemName"`
	Description string `json:"Description"`
	Location    string `json:"Location"`
	Time        string `json:"Time"`
}

func lossItemHandler(c *gin.Context) {
	var lossItems []LossItem
	rows, err := DB.Query("SELECT ItemID,Category, ItemName, Description, Location, DATE(Time) as Time FROM lossitem")
	if err != nil {
		logger.Error("Error querying lossitem table: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var lossItem LossItem
		var timeStr string
		if err := rows.Scan(&lossItem.ItemID, &lossItem.Category, &lossItem.ItemName, &lossItem.Description, &lossItem.Location, &timeStr); err != nil {
			logger.Error("Error scanning lossitem row: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
			return
		}
		parsedTime, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			logger.Error("Error parsing time: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
			return
		}
		lossItem.Time = parsedTime.Format("2006-01-02")
		lossItems = append(lossItems, lossItem)
	}

	if err = rows.Err(); err != nil {
		logger.Error("Error iterating over lossitem rows: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}
	logger.Info("Response: ", lossItems)

	c.JSON(http.StatusOK, lossItems)
}
func foundItemHandler(c *gin.Context) {
	var foundItems []foundItem
	rows, err := DB.Query("SELECT ItemID,Category, ItemName, Description, Location, DATE(Time) as Time FROM founditem")
	if err != nil {
		logger.Error("Error querying founditem table: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var foundItem foundItem
		var timeStr string
		if err := rows.Scan(&foundItem.ItemID, &foundItem.Category, &foundItem.ItemName, &foundItem.Description, &foundItem.Location, &timeStr); err != nil {
			logger.Error("Error scanning founditem row: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
			return
		}
		parsedTime, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			logger.Error("Error parsing time: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
			return
		}
		foundItem.Time = parsedTime.Format("2006-01-02")
		foundItems = append(foundItems, foundItem)
	}

	if err = rows.Err(); err != nil {
		logger.Error("Error iterating over founditem rows: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}
	logger.Info("Response: ", foundItems)

	c.JSON(http.StatusOK, foundItems)
}
func generateNextLossItemID(db *sql.DB) (string, error) {
	var maxID int
	err := db.QueryRow("SELECT MAX(CAST(SUBSTRING(ItemID, 2) AS UNSIGNED)) FROM lossitem").Scan(&maxID)
	if err != nil {
		return "", err
	}
	maxID++                                 // 增加1以获得下一个ID
	return fmt.Sprintf("L%03d", maxID), nil // 格式化为L001, L002, ...
}
func generateNextFoundItemID(db *sql.DB) (string, error) {
	var maxID int
	err := db.QueryRow("SELECT MAX(CAST(SUBSTRING(ItemID, 2) AS UNSIGNED)) FROM founditem").Scan(&maxID)
	if err != nil {
		return "", err
	}
	maxID++                                 // 增加1以获得下一个ID
	return fmt.Sprintf("F%03d", maxID), nil // 格式化为F001, F002, ...
}

func addLossItemHandler(c *gin.Context) {
	var lossItem struct {
		ID          string `json:"ID"`
		Category    string `json:"Category"`
		ItemName    string `json:"ItemName"`
		Description string `json:"Description"`
		Location    string `json:"Location"`
		Time        string `json:"Time"`
	}
	if err := c.ShouldBindJSON(&lossItem); err != nil {
		logger.Error("Failed to bind JSON: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nextItemID, err := generateNextLossItemID(DB)
	if err != nil {
		logger.Error("Error generating next item ID: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成ItemID失败"})
		return
	}

	var formattedTime string
	if lossItem.Time != "" {
		parsedTime, err := time.Parse(time.RFC3339, lossItem.Time)
		if err != nil {
			logger.Error("Error parsing time: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "时间格式错误"})
			return
		}
		formattedTime = parsedTime.Format("2006-01-02 15:04:05")
	} else {
		now := time.Now()
		formattedTime = now.Format("2006-01-02 15:04:05")
	}

	insertStmt := "INSERT INTO lossitem (ItemID, ID, Category, ItemName, Description, Location, Time) VALUES (?, ?, ?, ?, ?, ?, ?)"
	res, err := DB.Exec(insertStmt, nextItemID, lossItem.ID, lossItem.Category, lossItem.ItemName, lossItem.Description, lossItem.Location, formattedTime)
	if err != nil {
		logger.Error("Error inserting into lossitem table: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		logger.Error("Error affecting rows: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "插入数据失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "招领信息已提交"})
}

func addFoundItemHandler(c *gin.Context) {
	var foundItem struct {
		ID          string `json:"ID"`
		Category    string `json:"Category"`
		ItemName    string `json:"ItemName"`
		Description string `json:"Description"`
		Location    string `json:"Location"`
		Time        string `json:"Time"`
	}
	if err := c.ShouldBindJSON(&foundItem); err != nil {
		logger.Error("Failed to bind JSON: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nextItemID, err := generateNextFoundItemID(DB)
	if err != nil {
		logger.Error("Error generating next item ID: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成ItemID失败"})
		return
	}

	var formattedTime string
	if foundItem.Time != "" {
		parsedTime, err := time.Parse(time.RFC3339, foundItem.Time)
		if err != nil {
			logger.Error("Error parsing time: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "时间格式错误"})
			return
		}
		formattedTime = parsedTime.Format("2006-01-02 15:04:05")
	} else {
		now := time.Now()
		formattedTime = now.Format("2006-01-02 15:04:05")
	}

	insertStmt := "INSERT INTO founditem (ItemID, ID, Category, ItemName, Description, Location, Time) VALUES (?, ?, ?, ?, ?, ?, ?)"
	res, err := DB.Exec(insertStmt, nextItemID, foundItem.ID, foundItem.Category, foundItem.ItemName, foundItem.Description, foundItem.Location, formattedTime)
	if err != nil {
		logger.Error("Error inserting into founditem table: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		logger.Error("Error affecting rows: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "插入数据失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "招领信息已提交"})
}

func generateNextClaimFormID(db *sql.DB) (string, error) {
	var maxID sql.NullInt64
	err := db.QueryRow("SELECT MAX(CAST(SUBSTRING(ClaimFormID, 2) AS UNSIGNED)) FROM claimform").Scan(&maxID)
	if err != nil {
		if err == sql.ErrNoRows {
			// 如果表为空，从1开始
			maxID.Int64 = 0
		} else {
			return "", err
		}
	}

	// 检查是否为NULL并递增
	if maxID.Valid {
		maxID.Int64++
	} else {
		maxID.Int64 = 1
	}

	return fmt.Sprintf("CF%03d", maxID.Int64), nil
}
func confirmClaimHandler(c *gin.Context) {
	var updateReq struct {
		ItemID string `json:"ItemID"`
		ID     string `json:"ID"` // 申领人的ID
	}
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		logger.Println("Failed to bind JSON: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claimFormID, err := generateNextClaimFormID(DB)
	if err != nil {
		logger.Error("Error generating next claim form ID: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成申领单ID失败"})
		return
	}

	// 检查物品是否已经存在申领记录
	checkStmt := "SELECT COUNT(*) FROM claimform WHERE ItemID = ?"
	var isClaimed int
	err = DB.QueryRow(checkStmt, updateReq.ItemID).Scan(&isClaimed)
	if err != nil {
		logger.Error("Error checking if item has been claimed: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}

	if isClaimed > 0 {
		c.JSON(http.StatusOK, gin.H{"message": "该物品已被认领，请联系管理员进行确认！"})
		return
	}

	// 插入新的申领记录
	insertStmt := "INSERT INTO claimform (ClaimFormID, ItemID, ID, Status, ApplyTime) VALUES (?, ?, ?, '处理中', NOW())"
	res, err := DB.Exec(insertStmt, claimFormID, updateReq.ItemID, updateReq.ID)
	if err != nil {
		logger.Error("Error inserting into claimform table: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}

	affected, err := res.RowsAffected()
	if err != nil {
		logger.Error("Error checking rows affected: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查受影响行时出错"})
		return
	}

	if affected == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "未知错误"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "申领成功"})
	}
}
func getClaimFormsHandler(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Println("Failed to bind JSON: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 解析Token
	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte("ushjlwmwnwht"), nil
	})
	if err != nil || !token.Valid {
		logger.Errorf("invalid Token: %v", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的Token"})
		return
	}

	// 从Token的claims中获取用户ID
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.Error("Token解析错误")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Token解析错误"})
		return
	}
	userID := claims["id"].(string)
	logger.Info("userID:", userID)

	// 查询当前用户的认领单信息
	var claimForms []struct {
		ClaimFormID string `json:"ClaimFormID"`
		ItemID      string `json:"ItemID"`
		ApplyTime   string `json:"ApplyTime"`
		Status      string `json:"Status"`
		Category    string `json:"Category"`
		ItemName    string `json:"ItemName"`
	}
	rows, err := DB.Query("SELECT cf.ClaimFormID, li.ItemID, DATE_FORMAT(li.Time, '%Y-%m-%d %H:%i:%s') as ApplyTime, cf.Status, li.Category, li.ItemName FROM claimform cf INNER JOIN founditem li ON cf.ItemID = li.ItemID WHERE cf.ID = ?", userID)
	if err != nil {
		logger.Error("Error querying claimform table: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var claimForm struct {
			ClaimFormID string `json:"ClaimFormID"`
			ItemID      string `json:"ItemID"`
			ApplyTime   string `json:"ApplyTime"`
			Status      string `json:"Status"`
			Category    string `json:"Category"`
			ItemName    string `json:"ItemName"`
		}
		if err := rows.Scan(&claimForm.ClaimFormID, &claimForm.ItemID, &claimForm.ApplyTime, &claimForm.Status, &claimForm.Category, &claimForm.ItemName); err != nil {
			logger.Error("Error scanning claimform row: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
			return
		}
		claimForms = append(claimForms, claimForm)
	}

	if err = rows.Err(); err != nil {
		logger.Error("Error iterating over claimform rows: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}

	// 打印返回的信息
	logger.Info("Returning claim forms: ", claimForms)

	c.JSON(http.StatusOK, claimForms)
}
func complainHandler(c *gin.Context) {
	var complaint struct {
		ID       string `json:"ID"`
		UserID   string `json:"UserID"`
		Category string `json:"Category"`
		Reason   string `json:"Reason"`
	}
	if err := c.ShouldBindJSON(&complaint); err != nil {
		logger.Error("Failed to bind JSON: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var userCount int
	err := DB.QueryRow("SELECT COUNT(*) FROM user WHERE ID = ?", complaint.UserID).Scan(&userCount)
	if err != nil {
		logger.Error("Error checking user ID: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}
	if userCount == 0 {
		c.JSON(http.StatusOK, Response{Success: 0, Message: "该学号不存在,请仔细确认"})
		return
	}
	var maxID sql.NullInt64
	err = DB.QueryRow("SELECT MAX(CAST(SUBSTRING(ComplaintID, 2) AS UNSIGNED)) FROM complaint").Scan(&maxID)
	if err != nil && err != sql.ErrNoRows {
		logger.Error("Error querying max ComplaintID: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询最大ComplaintID失败"})
		return
	}
	if maxID.Valid {
		maxID.Int64++
	} else {
		maxID.Int64 = 1
	}
	nextComplaintID := fmt.Sprintf("CF%04d", maxID.Int64)
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	insertStmt := "INSERT INTO complaint (ComplaintID, ID, UserID, Category, Reason, Time) VALUES (?, ?, ?, ?, ?, ?)"
	res, err := DB.Exec(insertStmt, nextComplaintID, complaint.ID, complaint.UserID, complaint.Category, complaint.Reason, currentTime)
	if err != nil {
		logger.Error("Error inserting into complaint table: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		logger.Error("Error affecting rows: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "插入数据失败"})
		return
	}
	c.JSON(http.StatusOK, Response{Success: 1, Message: "投诉已提交"})
}

func complaintHandler(c *gin.Context) {
	var complaint struct {
		ID string `json:"ID"`
	}
	if err := c.ShouldBindJSON(&complaint); err != nil {
		logger.Printf("Failed to bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Printf("ID: %+v\n", complaint.ID)

	selectStmt := "SELECT UserID, Category, Reason, Time, Advice, Time2 FROM complaint WHERE ID = ?"
	rows, err := DB.Query(selectStmt, complaint.ID)
	if err != nil {
		logger.Printf("Error querying complaint table: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	var results []struct {
		UserID   string `json:"UserID"`
		Category string `json:"Category"` // 添加投诉类别字段
		Reason   string `json:"Reason"`
		Time     string `json:"Time"`
		Advice   string `json:"Advice"`
		Time2    string `json:"Time2"`
	}
	for rows.Next() {
		var result struct {
			UserID   string         `json:"UserID"`
			Category sql.NullString `json:"Category"` // 使用 sql.NullString 处理可能的 NULL 值
			Reason   string         `json:"Reason"`
			Time     string         `json:"Time"`
			Advice   sql.NullString `json:"-"` // 使用 "-" 忽略这个字段的序列化
			Time2    sql.NullString `json:"-"` // 使用 "-" 忽略这个字段的序列化
		}
		if err := rows.Scan(&result.UserID, &result.Category, &result.Reason, &result.Time, &result.Advice, &result.Time2); err != nil {
			logger.Printf("Error scanning rows: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取数据失败"})
			return
		}
		// 将 sql.NullString 转换为字符串，如果为 NULL，则转换为 "无"
		result.Advice.String = convertNullString(result.Advice.String)
		result.Time2.String = convertNullString(result.Time2.String)
		result.Advice.Valid = false // 因为我们已经转换为字符串，所以这里设置 Valid 为 false
		result.Time2.Valid = false  // 因为我们已经转换为字符串，所以这里设置 Valid 为 false

		// 将时间从 ISO 8601 格式转换为 "YYYY-MM-DD HH:MM:SS"
		timeLayout := "2006-01-02 15:04:05"
		timeFormat, _ := time.Parse(time.RFC3339, result.Time)
		formattedTime := timeFormat.Format(timeLayout)

		var formattedTime2 string
		if result.Time2.Valid {
			timeFormat2, _ := time.Parse(time.RFC3339, result.Time2.String)
			formattedTime2 = timeFormat2.Format(timeLayout)
		} else {
			formattedTime2 = "无"
		}

		// 将 Category 转换为字符串，如果为 NULL，则转换为 "无"
		category := convertNullString(result.Category.String)

		results = append(results, struct {
			UserID   string `json:"UserID"`
			Category string `json:"Category"`
			Reason   string `json:"Reason"`
			Time     string `json:"Time"`
			Advice   string `json:"Advice"`
			Time2    string `json:"Time2"`
		}{
			UserID:   result.UserID,
			Category: category,
			Reason:   result.Reason,
			Time:     formattedTime,
			Advice:   result.Advice.String,
			Time2:    formattedTime2,
		})
		fmt.Printf("Query Result: %+v\n", results[len(results)-1]) // 打印最后一个结果
	}
	if err := rows.Err(); err != nil {
		logger.Printf("Error reading rows: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取数据失败"})
		return
	}
	var data interface{}
	var message string
	if len(results) > 0 {
		data = results
		message = "投诉信息查询成功"
	} else {
		data = []struct {
			UserID   string `json:"UserID"`
			Category string `json:"Category"`
			Reason   string `json:"Reason"`
			Time     string `json:"Time"`
			Advice   string `json:"Advice"`
			Time2    string `json:"Time2"`
		}{} // 返回一个空数组
		message = "未找到投诉信息"
	}
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"data":    data,
	})
}

// 将 sql.NullString 转换为字符串，如果为 NULL，则转换为 "无"
func convertNullString(s string) string {
	if s == "" { // 检查字符串是否为空
		return "无"
	}
	return s
}

func ReclaimInfoHandler(c *gin.Context) {
	var reclaimInfo struct {
		ClaimFormID string `json:"claim_form_id"`
		ItemID      string `json:"item_id"`
		ApplyTime   string `json:"apply_time"`
		Status      string `json:"status"`
		ClaimerID   string `json:"claimer_id"`
		ClaimerName string `json:"claimer_name"`
		ClaimerTel  string `json:"claimer_telephone"`
		FinderID    string `json:"finder_id"`
		FinderName  string `json:"finder_name"`
		FinderTel   string `json:"finder_telephone"`
		Category    string `json:"category"`
		ItemName    string `json:"item_name"`
		Description string `json:"description"`
		Location    string `json:"location"`
		FoundTime   string `json:"found_time"`
	}

	query := `SELECT 
                    cf.ClaimFormID, 
                    cf.ItemID, 
                    DATE_FORMAT(cf.ApplyTime, '%Y-%m-%d %H:%i:%s') as ApplyTime, 
                    cf.Status, 
                    cf.ID as ClaimerID,
                    s.Name as ClaimerName, 
                    s.Telephone as ClaimerTel, 
                    fi.ID as FinderID, 
                    u.Name as FinderName, 
                    u.Telephone as FinderTel, 
                    fi.Category, 
                    fi.ItemName, 
                    fi.Description, 
                    fi.Location, 
                    DATE_FORMAT(fi.Time, '%Y-%m-%d %H:%i:%s') as FoundTime
                FROM claimform cf 
                INNER JOIN student s ON cf.ID = s.ID 
                INNER JOIN founditem fi ON cf.ItemID = fi.ItemID
                INNER JOIN student u ON fi.ID = u.ID`

	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询申领信息失败"})
		logger.Printf("查询申领信息失败: %v", err)
		return
	}
	defer rows.Close()

	var results []struct {
		ClaimFormID string `json:"claim_form_id"`
		ItemID      string `json:"item_id"`
		ApplyTime   string `json:"apply_time"`
		Status      string `json:"status"`
		ClaimerID   string `json:"claimer_id"`
		ClaimerName string `json:"claimer_name"`
		ClaimerTel  string `json:"claimer_telephone"`
		FinderID    string `json:"finder_id"`
		FinderName  string `json:"finder_name"`
		FinderTel   string `json:"finder_telephone"`
		Category    string `json:"category"`
		ItemName    string `json:"item_name"`
		Description string `json:"description"`
		Location    string `json:"location"`
		FoundTime   string `json:"found_time"`
	}

	for rows.Next() {
		if err := rows.Scan(&reclaimInfo.ClaimFormID, &reclaimInfo.ItemID, &reclaimInfo.ApplyTime, &reclaimInfo.Status, &reclaimInfo.ClaimerID, &reclaimInfo.ClaimerName, &reclaimInfo.ClaimerTel, &reclaimInfo.FinderID, &reclaimInfo.FinderName, &reclaimInfo.FinderTel, &reclaimInfo.Category, &reclaimInfo.ItemName, &reclaimInfo.Description, &reclaimInfo.Location, &reclaimInfo.FoundTime); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取数据失败"})
			logger.Printf("读取数据失败: %v", err)
			return
		}
		results = append(results, reclaimInfo)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取数据错误"})
		logger.Printf("读取数据错误: %v", err)
		return
	}

	var data interface{}
	var message string
	if len(results) > 0 {
		data = results
		message = "查询成功"
	} else {
		data = []struct {
			ClaimFormID string `json:"claim_form_id"`
			ItemID      string `json:"item_id"`
			ApplyTime   string `json:"apply_time"`
			Status      string `json:"status"`
			ClaimerID   string `json:"claimer_id"`
			ClaimerName string `json:"claimer_name"`
			ClaimerTel  string `json:"claimer_telephone"`
			FinderID    string `json:"finder_id"`
			FinderName  string `json:"finder_name"`
			FinderTel   string `json:"finder_telephone"`
			Category    string `json:"category"`
			ItemName    string `json:"item_name"`
			Description string `json:"description"`
			Location    string `json:"location"`
			FoundTime   string `json:"found_time"`
		}{}
		message = "未找到申领信息"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"data":    data,
	})
}

// ProcessReclaimHandler 处理申领单
func ProcessReclaimHandler(c *gin.Context) {
	var req struct {
		ClaimFormID string `json:"claim_form_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: 0, Message: "请求体格式错误"})
		return
	}

	// 更新claimform表的状态为已处理
	updateStmt := "UPDATE claimform SET Status = '已处理' WHERE ClaimFormID = ?"
	res, err := DB.Exec(updateStmt, req.ClaimFormID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: 0, Message: "更新失败"})
		logger.Printf("更新申领单状态失败: %v", err)
		return
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		c.JSON(http.StatusOK, Response{Success: 0, Message: "申领单不存在或已处理"})
	} else {
		c.JSON(http.StatusOK, Response{Success: 1, Message: "处理成功"})
	}
}

type NullString struct {
	sql.NullString
}

// MarshalJSON 实现 json.Marshaler 接口
func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte(`"无"`), nil
	}
	return []byte(`"` + ns.String + `"`), nil
}
func ComplaintInfoHandler(c *gin.Context) {
	var complaints []struct {
		ComplaintID string     `json:"complaint_id"`
		ID          string     `json:"id"`     // 投诉人
		AdmID       NullString `json:"adm_id"` // 使用 sql.NullString 处理可能的 NULL 值
		ClaimFormID NullString `json:"claim_form_id"`
		UserID      string     `json:"user_id"` // 被投诉者
		Category    string     `json:"category"`
		Reason      string     `json:"reason"`
		Time        string     `json:"time"`
		Advice      NullString `json:"advice"` // 使用 sql.NullString 处理可能的 NULL 值
		Time2       NullString `json:"time2"`  // 使用 sql.NullString 处理可能的 NULL 值
	}

	query := `SELECT ComplaintID, ID, Adm_ID, ClaimFormID, UserID, Category, Reason, DATE_FORMAT(Time, '%Y-%m-%d %H:%i:%s') as Time, Advice, Time2 FROM complaint`
	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询投诉信息失败"})
		logger.Printf("查询投诉信息失败: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var complaint struct {
			ComplaintID string     `json:"complaint_id"`
			ID          string     `json:"id"` // 投诉人
			AdmID       NullString `json:"adm_id"`
			ClaimFormID NullString `json:"claim_form_id"`
			UserID      string     `json:"user_id"` // 被投诉者
			Category    string     `json:"category"`
			Reason      string     `json:"reason"`
			Time        string     `json:"time"`
			Advice      NullString `json:"advice"`
			Time2       NullString `json:"time2"`
		}
		if err := rows.Scan(&complaint.ComplaintID, &complaint.ID, &complaint.AdmID, &complaint.ClaimFormID, &complaint.UserID, &complaint.Category, &complaint.Reason, &complaint.Time, &complaint.Advice, &complaint.Time2); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取数据失败"})
			logger.Printf("读取数据失败: %v", err)
			return
		}
		// 将 sql.NullString 转换为字符串，如果为 NULL，则转换为 "无"
		complaint.AdmID.String = convertNullString(complaint.AdmID.String)
		complaint.ClaimFormID.String = convertNullString(complaint.ClaimFormID.String)
		complaint.Advice.String = convertNullString(complaint.Advice.String)
		complaint.Time2.String = convertNullString(complaint.Time2.String)
		

		complaints = append(complaints, complaint)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取数据错误"})
		logger.Printf("读取数据错误: %v", err)
		return
	}
	logger.Print(complaints)

	c.JSON(http.StatusOK, gin.H{
		"data": complaints,
	})
}

func ProcessComplaintHandler(c *gin.Context) {
    var req struct {
		ID string `json:"admin_id"`
        ComplaintID string `json:"complaint_id"`
        Advice     string `json:"advice"`
        Time2      string `json:"time2"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请求体格式错误"})
        return
    }

    // 获取当前时间并格式化为 MySQL 接受的日期时间格式
    currentTime := time.Now().Format("2006-01-02 15:04:05")

    // 如果 Time2 是空的，使用当前时间
    if req.Time2 == "" {
        req.Time2 = currentTime
    }

    updateStmt := "UPDATE complaint SET Adm_ID = ?, Advice = ?, Time2 = ? WHERE ComplaintID = ?"
    res, err := DB.Exec(updateStmt, req.ID, req.Advice, req.Time2, req.ComplaintID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
        logger.Printf("更新投诉信息失败: %v", err)
        return
    }
    if affected, _ := res.RowsAffected(); affected == 0 {
        c.JSON(http.StatusOK, gin.H{"success": 0, "message": "投诉不存在或已处理"})
    } else {
        c.JSON(http.StatusOK, gin.H{"success": 1, "message": "处理成功"})
    }
}