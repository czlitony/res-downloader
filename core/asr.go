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

	gomp4 "github.com/abema/go-mp4"
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

// audioCodecType 音频编解码类型
type audioCodecType int

const (
	audioCodecAAC audioCodecType = iota
	audioCodecMP3
	audioCodecHEAAC // HE-AAC (SBR) - B站ASR不支持
	audioCodecUnknown
)

// detectAudioCodec 检测MP4中音频轨道的编解码类型
// 返回: (编解码类型, AAC profile用于ADTS头, 编解码名称)
// 注意: ADTS profile 只有 2 位 (0=Main, 1=LC, 2=SSR, 3=LTP)
func detectAudioCodec(mp4aInfo *gomp4.MP4AInfo) (audioCodecType, uint8, string) {
	if mp4aInfo == nil {
		return audioCodecAAC, 1, "aac (默认LC)"
	}

	// Object Type Indication (OTI) 标准值:
	// 0x40 = MPEG-4 Audio (AAC)
	// 0x66 = MPEG-2 AAC Main
	// 0x67 = MPEG-2 AAC LC
	// 0x68 = MPEG-2 AAC SSR
	// 0x69 = MPEG-2 Audio Layer III (MP3)
	// 0x6B = MPEG-1 Audio Layer III (MP3)
	oti := mp4aInfo.OTI

	switch oti {
	case 0x6B: // MPEG-1 Audio Layer III
		return audioCodecMP3, 0, "mp3 (MPEG-1 Layer III)"
	case 0x69: // MPEG-2 Audio Layer III
		return audioCodecMP3, 0, "mp3 (MPEG-2 Layer III)"
	case 0x66: // MPEG-2 AAC Main
		return audioCodecAAC, 0, "aac (MPEG-2 Main)"
	case 0x67: // MPEG-2 AAC LC
		return audioCodecAAC, 1, "aac (MPEG-2 LC)"
	case 0x68: // MPEG-2 AAC SSR
		return audioCodecAAC, 2, "aac (MPEG-2 SSR)"
	case 0x40: // MPEG-4 Audio (最常见的AAC)
		// AudOTI = Audio Object Type
		// 1=Main, 2=LC, 3=SSR, 4=LTP, 5=SBR(HE-AAC), 29=PS(HE-AACv2)
		// ADTS profile = AudOTI - 1, 但只有 2 位
		audOTI := mp4aInfo.AudOTI
		var profile uint8 = 1 // 默认 LC
		profileName := "LC"

		switch audOTI {
		case 1:
			profile = 0
			profileName = "Main"
		case 2:
			profile = 1
			profileName = "LC"
		case 3:
			profile = 2
			profileName = "SSR"
		case 4:
			profile = 3
			profileName = "LTP"
		case 5:
			// HE-AAC (SBR) - B站ASR不支持此格式
			globalLogger.Warn().Msgf("检测到 HE-AAC (SBR) 格式，B站ASR不支持")
			return audioCodecHEAAC, 1, "aac (HE-AAC/SBR - 不支持)"
		case 29:
			// HE-AACv2 (PS) - B站ASR不支持此格式
			globalLogger.Warn().Msgf("检测到 HE-AACv2 (PS) 格式，B站ASR不支持")
			return audioCodecHEAAC, 1, "aac (HE-AACv2/PS - 不支持)"
		default:
			profile = 1 // 未知类型默认用 LC
			profileName = fmt.Sprintf("AudOTI=%d (as LC)", audOTI)
		}
		globalLogger.Info().Msgf("AAC OTI=0x%02X, AudOTI=%d, ADTS profile=%d (%s)", oti, audOTI, profile, profileName)
		return audioCodecAAC, profile, fmt.Sprintf("aac (MPEG-4 %s)", profileName)
	default:
		// 未知 OTI，默认当作 AAC-LC 处理
		globalLogger.Warn().Msgf("未知的音频 OTI: 0x%02X，将尝试作为 AAC-LC 处理", oti)
		return audioCodecAAC, 1, fmt.Sprintf("aac (OTI=0x%02X as LC)", oti)
	}
}

// videoToAudio 从MP4容器中提取音频轨道（纯Go，无需ffmpeg）
// 支持的音频格式: AAC (各种profile), MP3
// 返回: (输出文件路径, 音频格式如"aac"/"mp3", error)
func videoToAudio(inputPath string) (string, string, error) {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return "", "", fmt.Errorf("打开视频文件失败: %v", err)
	}
	defer inputFile.Close()

	// 使用 abema/go-mp4 的 Probe 解析 MP4 结构
	info, err := gomp4.Probe(inputFile)
	if err != nil {
		return "", "", fmt.Errorf("解析MP4文件头失败: %v", err)
	}

	globalLogger.Info().Msgf("MP4文件信息: brand=%s, 时长=%d, 轨道数=%d",
		string(info.MajorBrand[:]), info.Duration, len(info.Tracks))

	// 找到音频轨道（CodecMP4A 包含 AAC 和 MP3）
	var audioTrack *gomp4.Track
	for _, t := range info.Tracks {
		if t.Codec == gomp4.CodecMP4A {
			audioTrack = t
			break
		}
	}

	if audioTrack == nil {
		// 列出所有轨道信息帮助调试
		var trackInfo []string
		for i, t := range info.Tracks {
			trackInfo = append(trackInfo, fmt.Sprintf("Track%d: codec=%d, encrypted=%v", i, t.Codec, t.Encrypted))
		}
		return "", "", fmt.Errorf("视频中未找到支持的音频轨道 (mp4a)。轨道列表: %v", trackInfo)
	}

	// 检测音频编解码类型
	codecType, aacProfile, codecName := detectAudioCodec(audioTrack.MP4A)

	// 检查是否为不支持的格式
	if codecType == audioCodecHEAAC {
		return "", "", fmt.Errorf("不支持的音频格式: %s。B站ASR仅支持AAC-LC和MP3格式，不支持HE-AAC(SBR)/HE-AACv2(PS)。建议使用其他工具将视频转换为普通AAC格式后再提取文案", codecName)
	}

	// 根据编解码类型确定输出格式
	var outputExt, audioFormat string
	switch codecType {
	case audioCodecMP3:
		outputExt = "_temp.mp3"
		audioFormat = "mp3"
	case audioCodecAAC:
		outputExt = "_temp.aac"
		audioFormat = "aac"
	default:
		outputExt = "_temp.aac"
		audioFormat = "aac"
	}

	channelCount := uint8(2)
	if audioTrack.MP4A != nil && audioTrack.MP4A.ChannelCount > 0 {
		channelCount = uint8(audioTrack.MP4A.ChannelCount)
	}
	freqIdx := aacFrequencyIndex(audioTrack.Timescale)

	globalLogger.Info().Msgf("音频轨道: trackID=%d, 编解码=%s, 采样率=%d, 声道=%d, samples=%d, chunks=%d",
		audioTrack.TrackID, codecName, audioTrack.Timescale, channelCount,
		len(audioTrack.Samples), len(audioTrack.Chunks))

	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + outputExt
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", "", fmt.Errorf("创建音频文件失败: %v", err)
	}
	defer outputFile.Close()

	// 构建 sample 偏移量表
	// Chunks 数组中每个 Chunk 包含: DataOffset (chunk在文件中的位置), SamplesPerChunk (该chunk包含的sample数)
	// Samples 数组中每个 Sample 包含: Size (帧大小)
	// 在 chunk 内，samples 是连续存储的
	type sampleInfo struct {
		offset int64
		size   uint32
	}
	var sampleOffsets []sampleInfo

	globalLogger.Info().Msgf("开始构建sample偏移量表: chunks=%d, totalSamples=%d",
		len(audioTrack.Chunks), len(audioTrack.Samples))

	sampleIdx := 0
	for chunkIdx, chunk := range audioTrack.Chunks {
		chunkOffset := int64(chunk.DataOffset)
		if chunkIdx < 3 {
			globalLogger.Debug().Msgf("Chunk %d: offset=%d, samplesPerChunk=%d",
				chunkIdx, chunkOffset, chunk.SamplesPerChunk)
		}
		for i := uint32(0); i < chunk.SamplesPerChunk && sampleIdx < len(audioTrack.Samples); i++ {
			sample := audioTrack.Samples[sampleIdx]
			sampleOffsets = append(sampleOffsets, sampleInfo{
				offset: chunkOffset,
				size:   sample.Size,
			})
			chunkOffset += int64(sample.Size)
			sampleIdx++
		}
	}

	if len(sampleOffsets) == 0 {
		os.Remove(outputPath)
		return "", "", fmt.Errorf("无法构建sample偏移量表")
	}

	globalLogger.Info().Msgf("偏移量表构建完成: %d samples, 第一个sample: offset=%d size=%d",
		len(sampleOffsets), sampleOffsets[0].offset, sampleOffsets[0].size)

	// 提取所有 sample
	hasData := false
	for idx, si := range sampleOffsets {
		if si.size == 0 {
			continue
		}

		// 定位并读取裸音频帧数据
		if _, err := inputFile.Seek(si.offset, io.SeekStart); err != nil {
			os.Remove(outputPath)
			return "", "", fmt.Errorf("seek sample %d 失败 (offset=%d): %v", idx, si.offset, err)
		}
		buf := make([]byte, si.size)
		if _, err := io.ReadFull(inputFile, buf); err != nil {
			os.Remove(outputPath)
			return "", "", fmt.Errorf("读取sample %d 失败 (offset=%d, size=%d): %v", idx, si.offset, si.size, err)
		}

		// AAC 需要添加 ADTS 头，MP3 直接写入原始数据
		if codecType == audioCodecAAC {
			adts := makeADTSHeader(aacProfile, freqIdx, channelCount, uint16(si.size))
			// 第一帧打印详细调试信息
			if idx == 0 {
				globalLogger.Info().Msgf("ADTS参数: profile=%d, freqIdx=%d, channels=%d, frameLen=%d",
					aacProfile, freqIdx, channelCount, si.size)
				globalLogger.Info().Msgf("ADTS头 (hex): %02X %02X %02X %02X %02X %02X %02X",
					adts[0], adts[1], adts[2], adts[3], adts[4], adts[5], adts[6])
				if len(buf) >= 4 {
					globalLogger.Info().Msgf("AAC帧前4字节 (hex): %02X %02X %02X %02X", buf[0], buf[1], buf[2], buf[3])
				}
			}
			outputFile.Write(adts)
		}
		outputFile.Write(buf)
		hasData = true
	}

	if !hasData {
		os.Remove(outputPath)
		return "", "", fmt.Errorf("未能从视频中提取到音频数据")
	}

	// 确保数据写入磁盘
	outputFile.Sync()

	// 检查输出文件大小
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		return "", "", fmt.Errorf("检查输出文件失败: %v", err)
	}
	if fileInfo.Size() < 1000 {
		globalLogger.Warn().Msgf("提取的音频文件过小: %d bytes，可能无效", fileInfo.Size())
	}

	globalLogger.Info().Msgf("音频提取完成: %s (格式: %s, 采样数: %d, 文件大小: %d bytes)", outputPath, audioFormat, len(sampleOffsets), fileInfo.Size())
	return outputPath, audioFormat, nil
}

// makeADTSHeader 生成7字节的ADTS头，用于封装裸AAC帧
// profile: ADTS profile (0=Main, 1=LC, 2=SSR, 3=LTP)
// freqIdx: 采样率索引 (见 aacFrequencyIndex)
// chanConf: 声道配置
// frameLen: 裸AAC帧长度（不含ADTS头）
func makeADTSHeader(profile, freqIdx, chanConf uint8, frameLen uint16) []byte {
	adtsLen := frameLen + 7
	h := make([]byte, 7)
	h[0] = 0xFF
	h[1] = 0xF1 // sync(4) + ID=0(MPEG-4) + layer=00 + protection_absent=1
	h[2] = (profile << 6) | (freqIdx << 2) | (chanConf >> 2)
	h[3] = ((chanConf & 0x3) << 6) | byte((adtsLen>>11)&0x3)
	h[4] = byte(adtsLen >> 3)
	h[5] = byte(adtsLen&0x7)<<5 | 0x1F // buffer fullness = 0x7FF (VBR)
	h[6] = 0xFC                         // buffer fullness low + numFrames-1=0
	return h
}

// aacFrequencyIndex 将采样率映射到ADTS频率索引
func aacFrequencyIndex(sampleRate uint32) uint8 {
	rates := [...]uint32{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000, 7350}
	for i, r := range rates {
		if sampleRate == r {
			globalLogger.Info().Msgf("采样率 %d Hz 匹配索引 %d", sampleRate, i)
			return uint8(i)
		}
	}
	// 如果不是标准采样率，尝试找最接近的
	globalLogger.Warn().Msgf("非标准采样率: %d Hz，尝试找最接近的值", sampleRate)
	for i, r := range rates {
		if sampleRate >= r {
			globalLogger.Info().Msgf("采样率 %d Hz 近似匹配索引 %d (%d Hz)", sampleRate, i, r)
			return uint8(i)
		}
	}
	globalLogger.Warn().Msgf("无法匹配采样率 %d Hz，使用默认索引 4 (44100 Hz)", sampleRate)
	return 4 // 默认44100
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

		globalLogger.Info().Msgf("查询状态: state=%d, remark=%s, result长度=%d", resultResp.Data.State, resultResp.Data.Remark, len(resultResp.Data.Result))

		// state: 0=停止, 1=运行中, 3=错误, 4=完成
		switch resultResp.Data.State {
		case 4: // 完成
			var asrResult ASRResult
			if err := json.Unmarshal([]byte(resultResp.Data.Result), &asrResult); err != nil {
				return nil, err
			}
			return &asrResult, nil
		case 3: // 错误
			remark := resultResp.Data.Remark
			if remark == "" {
				remark = "未知错误(可能是音频格式不支持或文件损坏)"
			}
			globalLogger.Error().Msgf("ASR失败完整响应: %s", string(respBody))
			return nil, fmt.Errorf("ASR识别失败: %s", remark)
		}

		// 等待3秒后继续查询（避免过于频繁）
		time.Sleep(3 * time.Second)
	}

	return nil, fmt.Errorf("查询超时")
}

// toText 将ASR结果转换为纯文本
func (asr *BcutASR) toText(result *ASRResult) string {
	var text strings.Builder
	for i, utterance := range result.Utterances {
		text.WriteString(utterance.Transcript)
		if i < len(result.Utterances)-1 {
			text.WriteString("\n")
		}
	}
	return text.String()
}

// SaveToFile 保存文本到文件
func SaveASRResultToFile(text, outputPath string) error {
	return os.WriteFile(outputPath, []byte(text), 0644)
}
