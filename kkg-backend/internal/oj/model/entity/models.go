package entity

import "time"

type User struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserAccount  string    `gorm:"column:userAccount" json:"userAccount"`
	UserPassword string    `gorm:"column:userPassword" json:"-"`
	UnionID      string    `gorm:"column:unionId" json:"unionId"`
	MpOpenID     string    `gorm:"column:mpOpenId" json:"mpOpenId"`
	UserName     string    `gorm:"column:userName" json:"userName"`
	UserAvatar   string    `gorm:"column:userAvatar" json:"userAvatar"`
	UserProfile  string    `gorm:"column:userProfile" json:"userProfile"`
	UserRole     string    `gorm:"column:userRole" json:"userRole"`
	CreateTime   time.Time `gorm:"column:createTime;autoCreateTime" json:"createTime"`
	UpdateTime   time.Time `gorm:"column:updateTime;autoUpdateTime" json:"updateTime"`
	IsDelete     int32     `gorm:"column:isDelete" json:"isDelete"`
}

func (User) TableName() string { return "user" }

type Question struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title       string    `gorm:"column:title" json:"title"`
	Content     string    `gorm:"column:content" json:"content"`
	Tags        string    `gorm:"column:tags" json:"tags"`
	Answer      string    `gorm:"column:answer" json:"answer"`
	SampleCase  string    `gorm:"column:sampleCase" json:"sampleCase"`
	SubmitNum   int32     `gorm:"column:submitNum" json:"submitNum"`
	AcceptedNum int32     `gorm:"column:acceptedNum" json:"acceptedNum"`
	JudgeCase   string    `gorm:"column:judgeCase" json:"judgeCase"`
	JudgeConfig string    `gorm:"column:judgeConfig" json:"judgeConfig"`
	ThumbNum    int32     `gorm:"column:thumbNum" json:"thumbNum"`
	FavourNum   int32     `gorm:"column:favourNum" json:"favourNum"`
	UserID      int64     `gorm:"column:userId" json:"userId"`
	CreateTime  time.Time `gorm:"column:createTime;autoCreateTime" json:"createTime"`
	UpdateTime  time.Time `gorm:"column:updateTime;autoUpdateTime" json:"updateTime"`
	IsDelete    int32     `gorm:"column:isDelete" json:"isDelete"`
}

func (Question) TableName() string { return "question" }

type QuestionSubmit struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Language   string    `gorm:"column:language" json:"language"`
	Code       string    `gorm:"column:code" json:"code"`
	JudgeInfo  string    `gorm:"column:judgeInfo" json:"judgeInfo"`
	Status     int32     `gorm:"column:status" json:"status"`
	QuestionID int64     `gorm:"column:questionId" json:"questionId"`
	UserID     int64     `gorm:"column:userId" json:"userId"`
	CreateTime time.Time `gorm:"column:createTime;autoCreateTime" json:"createTime"`
	UpdateTime time.Time `gorm:"column:updateTime;autoUpdateTime" json:"updateTime"`
	IsDelete   int32     `gorm:"column:isDelete" json:"isDelete"`
}

func (QuestionSubmit) TableName() string { return "question_submit" }

type QuestionSolutionPost struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	QuestionID int64     `gorm:"column:questionId;index:idx_question_post,priority:1;not null" json:"questionId"`
	PostID     int64     `gorm:"column:postId;index:idx_question_post,priority:2;not null" json:"postId"`
	UserID     int64     `gorm:"column:userId;not null" json:"userId"`
	CreateTime time.Time `gorm:"column:createTime;autoCreateTime" json:"createTime"`
	UpdateTime time.Time `gorm:"column:updateTime;autoUpdateTime" json:"updateTime"`
	IsDelete   int32     `gorm:"column:isDelete" json:"isDelete"`
}

func (QuestionSolutionPost) TableName() string { return "question_solution_post" }

type AgentSolutionTask struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	QuestionID     int64     `gorm:"column:questionId;index;not null" json:"questionId"`
	TriggerUserID  int64     `gorm:"column:triggerUserId;index;not null" json:"triggerUserId"`
	Status         string    `gorm:"column:status;size:32;not null;default:'pending'" json:"status"`
	Attempts       int32     `gorm:"column:attempts;not null;default:0" json:"attempts"`
	BlogPostID     int64     `gorm:"column:blogPostId;index;not null;default:0" json:"blogPostId"`
	BlogPostURL    string    `gorm:"column:blogPostUrl;size:512;not null;default:''" json:"blogPostUrl"`
	ModelName      string    `gorm:"column:modelName;size:128;not null;default:''" json:"modelName"`
	PromptSnapshot string    `gorm:"column:promptSnapshot;type:longtext" json:"promptSnapshot"`
	AnswerMarkdown string    `gorm:"column:answerMarkdown;type:longtext" json:"answerMarkdown"`
	AnswerCode     string    `gorm:"column:answerCode;type:longtext" json:"answerCode"`
	LastError      string    `gorm:"column:lastError;type:text" json:"lastError"`
	CreateTime     time.Time `gorm:"column:createTime;autoCreateTime" json:"createTime"`
	UpdateTime     time.Time `gorm:"column:updateTime;autoUpdateTime" json:"updateTime"`
	IsDelete       int32     `gorm:"column:isDelete" json:"isDelete"`
}

func (AgentSolutionTask) TableName() string { return "agent_solution_task" }

type Post struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title      string    `gorm:"column:title" json:"title"`
	Content    string    `gorm:"column:content" json:"content"`
	Tags       string    `gorm:"column:tags" json:"tags"`
	ThumbNum   int32     `gorm:"column:thumbNum" json:"thumbNum"`
	FavourNum  int32     `gorm:"column:favourNum" json:"favourNum"`
	UserID     int64     `gorm:"column:userId" json:"userId"`
	CreateTime time.Time `gorm:"column:createTime;autoCreateTime" json:"createTime"`
	UpdateTime time.Time `gorm:"column:updateTime;autoUpdateTime" json:"updateTime"`
	IsDelete   int32     `gorm:"column:isDelete" json:"isDelete"`
}

func (Post) TableName() string { return "post" }

type PostThumb struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PostID     int64     `gorm:"column:postId" json:"postId"`
	UserID     int64     `gorm:"column:userId" json:"userId"`
	CreateTime time.Time `gorm:"column:createTime;autoCreateTime" json:"createTime"`
	UpdateTime time.Time `gorm:"column:updateTime;autoUpdateTime" json:"updateTime"`
}

func (PostThumb) TableName() string { return "post_thumb" }

type PostFavour struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PostID     int64     `gorm:"column:postId" json:"postId"`
	UserID     int64     `gorm:"column:userId" json:"userId"`
	CreateTime time.Time `gorm:"column:createTime;autoCreateTime" json:"createTime"`
	UpdateTime time.Time `gorm:"column:updateTime;autoUpdateTime" json:"updateTime"`
}

func (PostFavour) TableName() string { return "post_favour" }
