package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mp4 "github.com/yapingcat/gomedia/go-mp4"
)

const (
	API_REQ_UPLOAD    = "https://member.bilibili.com/x/bcut/rubick-interface/resource/create"
	API_COMMIT_UPLOAD = "https://member.bilibili.com/x/bcut/rubick-interface/resource/create/complete"
	API_CREATE_TASK   = "https://member.bilibili.com/x/bcut/rubick-interface/task"
	API_QUERY_RESULT  = "https://member.bilibili.com/x/bcut/rubick-interface/task/result"
)

type BcutASR struct {
	AudioPath   string
	AudioFormat string // 实际音频格式: mp3, aac, wav, flac 等
	ResourceID  string
	DownloadURL string
	TaskID      string
	Cookie      string // B站Cookie，包含SESSDATA
	UploadID    string // 上传ID
	InBossKey   string // 上传密钥
	Etags       []string // 分片ETag列表
	client      *http.Client
}

type ASRUtterance struct {
	Transcript string `json:"transcript"`
	StartTime  int    `json:"start_time"`
	EndTime    int    `json:"end_time"`
}

type ASRResult struct {
	Utterances []ASRUtterance `json:"utterances"`
}

func NewBcutASR(audioPath string) *BcutASR {
	return &BcutASR{
		AudioPath: audioPath,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// setHeaders 设置请求头（与AsrTools保持一致）
func (asr *BcutASR) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Bilibili/1.0.0 (https://www.bilibili.com)")
	req.Header.Set("Content-Type", "application/json")
}

// Run 执行完整的ASR流程
func (asr *BcutASR) Run() (string, error) {
	globalLogger.Info().Msg("开始ASR识别: " + asr.AudioPath)

	// 0. 如果是视频文件，先用ffmpeg转成mp3
	audioPath := asr.AudioPath
	audioExts := map[string]bool{".mp3": true, ".wav": true, ".flac": true, ".m4a": true, ".ogg": true, ".aac": true, ".wma": true}
	ext := strings.ToLower(filepath.Ext(asr.AudioPath))
	if !audioExts[ext] {
		globalLogger.Info().Msg("步骤0: 视频转音频")
		extractedPath, audioFmt, err := videoToAudio(asr.AudioPath)
		if err != nil {
			return "", fmt.Errorf("视频转音频失败: %v", err)
		}
		audioPath = extractedPath
		asr.AudioFormat = audioFmt
	} else {
		// 原始就是音频文件，取扩展名（去掉点号）
		asr.AudioFormat = strings.TrimPrefix(ext, ".")
	}

	// 1. 上传音频文件
	globalLogger.Info().Msg("步骤1: 上传音频文件")
	asr.AudioPath = audioPath
	if err := asr.upload(); err != nil {
		globalLogger.Err(err)
		return "", fmt.Errorf("上传失败: %v", err)
	}

	// 2. 创建识别任务
	globalLogger.Info().Msg("步骤2: 创建识别任务, ResourceID: " + asr.ResourceID)
	if err := asr.createTask(); err != nil {
		globalLogger.Err(err)
		return "", fmt.Errorf("创建任务失败: %v", err)
	}

	// 3. 轮询查询结果
	globalLogger.Info().Msg("步骤3: 查询结果, TaskID: " + asr.TaskID)
	result, err := asr.pollResult()
	if err != nil {
		globalLogger.Err(err)
		return "", fmt.Errorf("查询结果失败: %v", err)
	}

	// 4. 转换为纯文本
	text := asr.toText(result)
	globalLogger.Info().Msgf("ASR识别完成, 文本长度: %d", len(text))
	return text, nil
}

// videoToAudio 从MP4容器中提取音频轨道（纯Go，无需ffmpeg）
// 返回: (输出文件路径, 音频格式如"aac"/"mp3", error)
func videoToAudio(inputPath string) (string, string, error) {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return "", "", fmt.Errorf("打开视频文件失败: %v", err)
	}
	defer inputFile.Close()

	demuxer := mp4.CreateMp4Demuxer(inputFile)

	// 读取所有轨道信息
	tracks, err := demuxer.ReadHead()
	if err != nil {
		return "", "", fmt.Errorf("解析MP4文件头失败: %v", err)
	}

	// 找到音频轨道（优先AAC，其次MP3/OPUS）
	var audioTrack *mp4.TrackInfo
	for i := range tracks {
		t := &tracks[i]
		if t.Cid == mp4.MP4_CODEC_AAC || t.Cid == mp4.MP4_CODEC_MP3 || t.Cid == mp4.MP4_CODEC_OPUS {
			audioTrack = t
			if t.Cid == mp4.MP4_CODEC_AAC {
				break // 优先AAC
			}
		}
	}

	if audioTrack == nil {
		return "", "", fmt.Errorf("视频中未找到音频轨道")
	}

	globalLogger.Info().Msgf("找到音频轨道: codec=%d, sampleRate=%d, channels=%d, trackId=%d",
		audioTrack.Cid, audioTrack.SampleRate, audioTrack.ChannelCount, audioTrack.TrackId)

	// 根据编码类型决定输出文件后缀和格式名
	var outputExt string
	var audioFormat string
	switch audioTrack.Cid {
	case mp4.MP4_CODEC_MP3:
		outputExt = "_temp.mp3"
		audioFormat = "mp3"
	default:
		outputExt = "_temp.aac"
		audioFormat = "aac"
	}
	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + outputExt
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", "", fmt.Errorf("创建音频文件失败: %v", err)
	}
	defer outputFile.Close()

	// 提取音频数据
	// gomedia的ReadPacket返回的AAC数据已包含ADTS头，直接写入即可
	audioTrackId := audioTrack.TrackId
	hasData := false
	for {
		pkg, err := demuxer.ReadPacket()
		if err != nil {
			break // io.EOF 或其他错误表示读取结束
		}
		if pkg.TrackId != audioTrackId {
			continue
		}
		outputFile.Write(pkg.Data)
		hasData = true
	}

	if !hasData {
		os.Remove(outputPath)
		return "", "", fmt.Errorf("未能从视频中提取到音频数据")
	}

	globalLogger.Info().Msgf("音频提取完成: %s (格式: %s)", outputPath, audioFormat)
	return outputPath, audioFormat, nil
}

// upload 上传音频文件
func (asr *BcutASR) upload() error {
	// 读取整个文件到内存
	fileData, err := os.ReadFile(asr.AudioPath)
	if err != nil {
		return err
	}

	// 1. 请求上传（使用form表单，字段名蛇形命名，与Python参考保持一致）
	audioFmt := asr.AudioFormat
	if audioFmt == "" {
		audioFmt = "mp3" // 默认mp3
	}
	formValues := url.Values{}
	formValues.Set("type", "2")
	formValues.Set("name", "audio."+audioFmt)
	formValues.Set("size", strconv.Itoa(len(fileData)))
	formValues.Set("resource_file_type", audioFmt)
	formValues.Set("model_id", "7")
	globalLogger.Info().Msgf("请求上传, body: %s", formValues.Encode())
	req, err := http.NewRequest("POST", API_REQ_UPLOAD, strings.NewReader(formValues.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Bilibili/1.0.0 (https://www.bilibili.com)")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := asr.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var uploadResp struct {
		Code int `json:"code"`
		Data struct {
			ResourceID    string   `json:"resource_id"`
			UploadID      string   `json:"upload_id"`
			UploadURLs    []string `json:"upload_urls"`
			PerUploadSize int64    `json:"per_size"`
			DownloadURL   string   `json:"download_url"`
			InBossKey     string   `json:"in_boss_key"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return err
	}

	if uploadResp.Code != 0 {
		bodyBytes, _ := json.Marshal(uploadResp)
		return fmt.Errorf("请求上传失败, code: %d, response: %s", uploadResp.Code, string(bodyBytes))
	}

	asr.ResourceID = uploadResp.Data.ResourceID
	asr.DownloadURL = uploadResp.Data.DownloadURL
	asr.UploadID = uploadResp.Data.UploadID
	asr.InBossKey = uploadResp.Data.InBossKey
	perSize := uploadResp.Data.PerUploadSize

	globalLogger.Info().Msgf("请求上传成功, ResourceID: %s, UploadURLs: %d, PerSize: %d, FileSize: %d",
		asr.ResourceID, len(uploadResp.Data.UploadURLs), perSize, len(fileData))

	// 2. 分片上传文件内容（与AsrTools一致：按per_size分片，每片一个URL）
	asr.Etags = []string{}
	uploadClient := &http.Client{Timeout: 300 * time.Second}

	for i, uploadURL := range uploadResp.Data.UploadURLs {
		// 计算当前分片的数据范围
		start := int64(i) * perSize
		end := start + perSize
		if end > int64(len(fileData)) {
			end = int64(len(fileData))
		}
		chunk := fileData[start:end]

		globalLogger.Info().Msgf("上传分片 %d/%d, 大小: %d bytes", i+1, len(uploadResp.Data.UploadURLs), len(chunk))

		uploadReq, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(chunk))
		if err != nil {
			return fmt.Errorf("创建分片上传请求失败: %v", err)
		}

		uploadHttpResp, err := uploadClient.Do(uploadReq)
		if err != nil {
			return fmt.Errorf("上传分片 %d 失败: %v", i+1, err)
		}

		if uploadHttpResp.StatusCode != 200 && uploadHttpResp.StatusCode != 201 {
			respBody, _ := io.ReadAll(uploadHttpResp.Body)
			uploadHttpResp.Body.Close()
			return fmt.Errorf("上传分片 %d 失败, status: %d, response: %s", i+1, uploadHttpResp.StatusCode, string(respBody))
		}

		// 获取ETag
		etag := uploadHttpResp.Header.Get("Etag")
		if etag == "" {
			etag = uploadHttpResp.Header.Get("ETag")
		}
		asr.Etags = append(asr.Etags, etag)
		uploadHttpResp.Body.Close()

		globalLogger.Info().Msgf("分片 %d 上传成功, ETag: %s", i+1, etag)
	}

	// 3. 提交上传（使用form表单，字段名蛇形命名）
	commitValues := url.Values{}
	commitValues.Set("in_boss_key", asr.InBossKey)
	commitValues.Set("resource_id", asr.ResourceID)
	commitValues.Set("etags", strings.Join(asr.Etags, ","))
	commitValues.Set("upload_id", asr.UploadID)
	commitValues.Set("model_id", "7")
	globalLogger.Info().Msgf("提交上传, body: %s", commitValues.Encode())
	commitReq, err := http.NewRequest("POST", API_COMMIT_UPLOAD, strings.NewReader(commitValues.Encode()))
	if err != nil {
		return err
	}
	commitReq.Header.Set("User-Agent", "Bilibili/1.0.0 (https://www.bilibili.com)")
	commitReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	commitResp, err := asr.client.Do(commitReq)
	if err != nil {
		return err
	}
	defer commitResp.Body.Close()

	// 读取完整响应体用于解析和调试
	commitRespBody, err := io.ReadAll(commitResp.Body)
	if err != nil {
		return fmt.Errorf("读取提交上传响应失败: %v", err)
	}
	globalLogger.Info().Msgf("提交上传响应: %s", string(commitRespBody))

	var commitResult struct {
		Code int `json:"code"`
		Data struct {
			DownloadURL string `json:"download_url"`
			ResourceID  string `json:"resource_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(commitRespBody, &commitResult); err != nil {
		return fmt.Errorf("解析提交上传响应失败: %v", err)
	}
	if commitResult.Code != 0 {
		return fmt.Errorf("提交上传失败, code: %d, response: %s", commitResult.Code, string(commitRespBody))
	}
	// 使用commit返回的download_url（这是最终的URL）
	if commitResult.Data.DownloadURL != "" {
		asr.DownloadURL = commitResult.Data.DownloadURL
	}

	globalLogger.Info().Msgf("上传完成, DownloadURL: %s", asr.DownloadURL)
	return nil
}

// createTask 创建识别任务
func (asr *BcutASR) createTask() error {
	taskData := map[string]interface{}{
		"resource": asr.DownloadURL,
		"model_id": "7",
	}

	taskBody, _ := json.Marshal(taskData)
	globalLogger.Info().Msgf("创建任务请求, body: %s", string(taskBody))
	req, err := http.NewRequest("POST", API_CREATE_TASK, bytes.NewBuffer(taskBody))
	if err != nil {
		return err
	}
	asr.setHeaders(req)

	resp, err := asr.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 读取完整响应体用于调试
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取创建任务响应失败: %v", err)
	}
	globalLogger.Info().Msgf("创建任务响应: %s", string(respBody))

	var taskResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &taskResp); err != nil {
		return fmt.Errorf("解析创建任务响应失败: %v, body: %s", err, string(respBody))
	}

	if taskResp.Code != 0 {
		return fmt.Errorf("创建任务失败, code: %d, message: %s, response: %s", taskResp.Code, taskResp.Message, string(respBody))
	}

	asr.TaskID = taskResp.Data.TaskID
	globalLogger.Info().Msgf("创建任务成功, TaskID: %s", asr.TaskID)
	return nil
}

// pollResult 轮询查询结果
func (asr *BcutASR) pollResult() (*ASRResult, error) {
	for i := 0; i < 500; i++ {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s?model_id=7&task_id=%s", API_QUERY_RESULT, asr.TaskID), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Bilibili/1.0.0 (https://www.bilibili.com)")

		resp, err := asr.client.Do(req)
		if err != nil {
			return nil, err
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var resultResp struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				State  int    `json:"state"`
				Remark string `json:"remark"`
				Result string `json:"result"`
			} `json:"data"`
		}

		if err := json.Unmarshal(respBody, &resultResp); err != nil {
			return nil, fmt.Errorf("解析查询响应失败: %v, body: %s", err, string(respBody))
		}

		if resultResp.Code != 0 {
			return nil, fmt.Errorf("查询结果失败, code: %d, message: %s", resultResp.Code, resultResp.Message)
		}

		globalLogger.Info().Msgf("查询状态: state=%d, remark=%s", resultResp.Data.State, resultResp.Data.Remark)

		// state: 0=停止, 1=运行中, 3=错误, 4=完成
		switch resultResp.Data.State {
		case 4: // 完成
			var asrResult ASRResult
			if err := json.Unmarshal([]byte(resultResp.Data.Result), &asrResult); err != nil {
				return nil, err
			}
			return &asrResult, nil
		case 3: // 错误
			return nil, fmt.Errorf("ASR识别失败: %s", resultResp.Data.Remark)
		}

		// 等待3秒后继续查询（避免过于频繁）
		time.Sleep(3 * time.Second)
	}

	return nil, fmt.Errorf("查询超时")
}

// toText 将ASR结果转换为纯文本
func (asr *BcutASR) toText(result *ASRResult) string {
	var text strings.Builder
	for _, utterance := range result.Utterances {
		text.WriteString(utterance.Transcript)
	}
	return text.String()
}

// SaveToFile 保存文本到文件
func SaveASRResultToFile(text, outputPath string) error {
	return os.WriteFile(outputPath, []byte(text), 0644)
}
