//
//  Agora Media SDK
//
//  Created by FanYuanYuan in 2022-05.
//  Copyright (c) 2022 Agora IO. All rights reserved.
//

#pragma once

#include "AgoraRefPtr.h"
#include "IAgoraMediaPlayer.h"
namespace agora {
namespace rtc {

#define MccVendorID MusicContentCenterVendorID
/**
 * @brief Enum for Music Content Center Vendor IDs.
 */
typedef enum {
    kMusicContentCenterVendorDefault = 1, ///< Default vendor
    kMusicContentCenterVendor2 = 2,       ///< Vendor 2
} MusicContentCenterVendorID;

/**
 * Modes for playing songs.
 */
typedef enum
{
    /**
     * 0: The music player is in the origin mode, which means playing the original song.
     */
    kMusicPlayModeOriginal = 0,

    /**
     * 1: The music player is in the accompany mode, which means playing the accompaniment only.
     */
    kMusicPlayModeAccompany = 1,
    
    /**
     * 2: The music player is in the lead sing mode, which means playing the lead vocals.
     */
    kMusicPlayModeLeadSing = 2,

} MusicPlayMode;

/**
 * The state of the music content center.
 */
typedef enum {
    kMusicContentCenterStatePreloadOk = 0,
    kMusicContentCenterStatePreloadFailed = 1,
    kMusicContentCenterStatePreloading = 2,
    kMusicContentCenterStatePreloadRemoved = 3,
    kMusicContentCenterStateStartScoreCompleted = 4,
    kMusicContentCenterStateStartScoreFailed = 5,
} MusicContentCenterState;

/**
 * The reason for the music content center state.
 */
typedef enum
{
    /**
     * 0: No error occurs and request succeeds.
     */
    kMusicContentCenterReasonOk = 0,
    /**
     * 1: A general error occurs.
     */
    kMusicContentCenterReasonError = 1,
    /**
     * 2: The gateway error. There are several possible reasons:
     *  - Token is expired. Check if your token is expired.
     *  - Token is invalid. Check the type of token you passed in.
     *  - Network error. Check your network.
     */
    kMusicContentCenterReasonGateway = 2,
    /**
     * 3: Permission and resource error. There are several possible reasons:
     *  - Your appid may not have the mcc permission. Please contact technical support 
     *  - The resource may not exist. Please contact technical support
     */
    kMusicContentCenterReasonPermissionAndResource = 3,
    /**
     * 4: Internal data parse error. Please contact technical support
     */
    kMusicContentCenterReasonInternalDataParse = 4,
    /**
     * 5: Music loading error. Please contact technical support
     */
    kMusicContentCenterReasonMusicLoading = 5,
    /**
     * 6: Music decryption error. Please contact technical support
     */
    kMusicContentCenterReasonMusicDecryption = 6, 
    /**
     * 7: Http internal error. Please retry later.
     */
    kMusicContentCenterReasonHttpInternalError = 7, 
} MusicContentCenterStateReason;

typedef struct 
{
    /**
     * Name of the music chart
     */
    const char* chartName;
    /**
     * Id of the music chart, which is used to get music list
     */
    int32_t id;
} MusicChartInfo;

enum MUSIC_CACHE_STATUS_TYPE {
    /**
     * 0: Music is already cached.
     */
    MUSIC_CACHE_STATUS_TYPE_CACHED = 0,
    /**
     * 1: Music is being cached.
     */
    MUSIC_CACHE_STATUS_TYPE_CACHING = 1,
    /**
     * 2: Music is not cached.
     */
    MUSIC_CACHE_STATUS_TYPE_NO_CACHED = 2,
    /**
     * 3: Music is no resource.
     */
    MUSIC_CACHE_STATUS_TYPE_NO_RESOURCE = 3
};

struct MusicCacheInfo {
    /**
     * The songCode of music.
     */
    int64_t songCode;
    /**
     * The cache status of the music.
     */
    MUSIC_CACHE_STATUS_TYPE musicStatus = MUSIC_CACHE_STATUS_TYPE_NO_CACHED;
    /**
     * The cache status of the lyric.
     */
    MUSIC_CACHE_STATUS_TYPE lyricStatus = MUSIC_CACHE_STATUS_TYPE_NO_CACHED;
};

class MusicChartCollection : public RefCountInterface {
public:
    virtual int getCount() = 0;
    virtual MusicChartInfo* get(int index) = 0;
protected:
    virtual ~MusicChartCollection() = default;
};

struct MvProperty
{
    /**
     * The resolution of the mv
     */
    const char* resolution;
    /**
     * The bandwidth of the mv
     */
    const char* bandwidth;
};

struct ClimaxSegment
{
    /**
     * The start time of climax segment
     */
    int32_t startTimeMs;
    /**
     * The end time of climax segment
     */
    int32_t endTimeMs;
};

struct Music
{
    /**
     * The songCode of music
     */
    int64_t songCode;
    /**
     * The name of music
     */
    const char* name;
    /**
     * The singer of music
     */
    const char* singer;
    /**
     * The poster url of music
     */
    const char* poster;
    /**
     * The release time of music
     */
    const char* releaseTime;
    /**
     * The duration (in seconds) of music
     */
    int32_t durationS;
    /**
     * The type of music
     * 1, mp3 with instrumental accompaniment and original
     * 2, mp3 only with instrumental accompaniment
     * 3, mp3 only with original
     * 4, mp4 with instrumental accompaniment and original
     * 5, mv only
     * 6, new type mp4 with instrumental accompaniment and original
     * detail at document of music media center
     */
    int32_t type;
    /**
     * The pitch type of music. 
     * 1, xml lyric has pitch
     * 2, lyric has no pitch
     */
    int32_t pitchType;
    /**
     * The number of lyrics available for the music
     */
    int32_t lyricCount;
    /**
     * The lyric list of music
     * 0, xml
     * 1, lrc
     */
    int32_t* lyricList;
    /**
     * The number of climax segments of the music
     */
    int32_t climaxSegmentCount;
    /**
     * The climax segment list of music
     */
    ClimaxSegment* climaxSegmentList;
    /**
     * The number of mv of the music
     * If this value is greater than zero, it means the current music has MV resource
     */
    int32_t mvPropertyCount;
    /**
     * The mv property list of music
     */
    MvProperty* mvPropertyList;
};

class MusicCollection : public RefCountInterface {
public:
    virtual int getCount() = 0;
    virtual int getTotal() = 0;
    virtual int getPage() = 0;
    virtual int getPageSize() = 0;
    virtual Music* getMusic(int32_t index) = 0;
protected:
    virtual ~MusicCollection() = default;
};

typedef enum { 
    /**
     * 0: xml
     */
    kLyricSourceXml = 0, 
    /**
     * 1: lrc
     */
    kLyricSourceLrc = 1, 
    /**
     * 2: lrc with pitches
     */
    kLyricSourceLrcWithPitches = 2,
    /**
     * 3: krc
     */
    kLyricSourceKrc = 3,
} LyricSourceType;

// The level of the score, the higher the level, the harder it is
typedef enum {
    kScoreLevel1 = 1,
    kScoreLevel2 = 2,
    kScoreLevel3 = 3,
    kScoreLevel4 = 4,
    kScoreLevel5 = 5,
} ScoreLevel;


class IWord: public RefCountInterface {
public:
  /**
   * @brief Gets the start time of the word in milliseconds.
   * @return The start time of the word in milliseconds.
   */
  virtual int getBegin() = 0;

  /**
   * @brief Gets the duration of the word in milliseconds.
   * @return The duration of the word in milliseconds.
   */
  virtual int getDuration() = 0;

  /**
   * @brief Gets the reference pitch of the word.
   * @return The reference pitch of the word.
   */
  virtual double getRefPitch() = 0;

  /**
   * @brief Gets the content of the word.
   * @return The content of the word.
   */
  virtual const char* getWord() = 0;

  /**
   * @brief Gets the score of the word.
   * @return The score of the word.
   */
  virtual int getScore() = 0;

protected:
  /**
   * @brief Destructor for IWord.
   */
  virtual ~IWord() = default;
};

class ISentence: public RefCountInterface {
public:
  /**
   * @brief Gets the content of the sentence.
   * @return The content of the sentence.
   */
  virtual const char* getContent() = 0;

  /**
   * @brief Gets the start time of the sentence in milliseconds.
   * @return The start time of the sentence in milliseconds.
   */
  virtual int getBegin() = 0;

  /**
   * @brief Gets the duration of the sentence in milliseconds.
   * @return The duration of the sentence in milliseconds.
   */
  virtual int getDuration() = 0;

  /**
   * @brief Gets a word in the sentence by index.
   * @param index The index of the word.
   * @return A reference to the word at the specified index.
   */
  virtual agora_refptr<IWord> getWord(int32_t index) = 0;

  /**
   * @brief Gets the number of words in the sentence.
   * @return The number of words in the sentence.
   */
  virtual int getWordCount() = 0;

  /**
   * @brief Gets the score of the sentence.
   * @return The score of the sentence.
   */
  virtual int getScore() = 0;

protected:
  /**
   * @brief Destructor for ISentence.
   */
  virtual ~ISentence() = default;
};

class ILyricInfo: public RefCountInterface {
public:
  /**
   * @brief Gets the name of the song.
   * @return The name of the song.
   */
  virtual const char* getName() = 0;

  /**
   * @brief Gets the singer of the song.
   * @return The singer of the song.
   */
  virtual const char* getSinger() = 0;

  /**
   * @brief Gets the end position of the prelude in milliseconds.
   * @return The end position of the prelude in milliseconds.
   */
  virtual int getPreludeEndPosition() = 0;

  /**
   * @brief Gets the duration of the song in milliseconds.
   * @return The duration of the song in milliseconds.
   */
  virtual int getDuration() = 0;

  /**
   * @brief Checks if the song has pitch information.
   * @return True if the song has pitch information, false otherwise.
   */
  virtual bool getHasPitch() = 0;

  /**
   * @brief Gets the source type of the lyrics.
   * @return The source type of the lyrics.
   */
  virtual LyricSourceType getSourceType() = 0;

  /**
   * @brief Gets a sentence in the song by index.
   * @param index The index of the sentence.
   * @return A reference to the sentence at the specified index.
   */
  virtual agora_refptr<ISentence> getSentence(int32_t index) = 0;

  /**
   * @brief Gets the number of sentences in the song.
   * @return The number of sentences in the song.
   */
  virtual int getSentenceCount() = 0;

protected:
  /**
   * @brief Destructor for ILyricInfo.
   */
  virtual ~ILyricInfo() = default;
};

struct RawScoreData {
    /**
     * The progress of the song in milliseconds.
     */
    int progressInMs;
    /**
     * The pitch of the speaker.
     */
    float speakerPitch = 0;
    /**
     * The pitch score of the speaker in the current progress.
     */
    float pitchScore = 0;
};

struct LineScoreData {
    /**
     * The progress of the song in milliseconds.
     */
    int progressInMs;
    /**
     * The index of the line.
     */          
    int index;
    /**
     * The number of lines in the song.
     */
    int totalLines;
    /**
     * The pitch score of the line.
     */
    float pitchScore;
    /**
     * The cumulative pitch score of the lines.
     */
    float cumulativePitchScore;
    /**
     * The energy score of the lines.
     */
    float energyScore;
};

struct CumulativeScoreData {
    /**
     * The progress of the song in milliseconds.
     */
    int progressInMs;
    /**
     * The cumulative pitch score of the lines.
     */
    float cumulativePitchScore;
    /**
     * The energy score of the lines.
     */
    float energyScore;
};

enum ChargeMode {
    /**
     * 1: Monthly charge mode
     */
    kChargeModeMonthly = 1,
    /**
     * 2: Charge once mode
     */
    kChargeModeOnce = 2,
};


class IScoreEventHandler {
public:
    /**
     * The callback when the pitch of the speaker is detected.
     */
    virtual void onPitch(int64_t songCode, RawScoreData &rawScoreData) = 0;
    /**
     * The callback when the score of the line is detected.
     */
    virtual void onLineScore(int64_t songCode, LineScoreData &lineScoreData) = 0;
        
    virtual ~IScoreEventHandler() {};
};


class IMusicContentCenterEventHandler {
public:
    /**
     * The music chart result callback; occurs when getMusicCharts method is called.
     * 
     * @param requestId The request id is same as that returned by getMusicCharts.
     * @param result The result of music chart collection
     * @param reason The status of the request. See MusicContentCenterStateReason
     */
    virtual void onMusicChartsResult(const char* requestId, agora_refptr<MusicChartCollection> result, MusicContentCenterStateReason reason) = 0;

    /**
     * Music collection, occurs when getMusicCollectionByMusicChartId or searchMusic method is called.
     * 
     * @param requestId The request id is same as that returned by getMusicCollectionByMusicChartId or searchMusic
     * @param result The result of music collection
     * @param reason The status of the request. See MusicContentCenterStateReason
     */
    virtual void onMusicCollectionResult(const char* requestId, agora_refptr<MusicCollection> result, MusicContentCenterStateReason reason) = 0;

    /**
     * Lyric url callback of getLyric, occurs when getLyric is called
     * 
     * @param requestId The request id is same as that returned by getLyric
     * @param songCode Song code
     * @param lyricUrl  The lyric url of this music
     * @param reason The status of the request. See MusicContentCenterStateReason
     */
    virtual void onLyricResult(const char* requestId, int64_t internalSongCode, const char* payload, MusicContentCenterStateReason reason) = 0;


    virtual void onLyricInfoResult(const char* requestId, int64_t songCode, agora_refptr<ILyricInfo> lyricInfo, MusicContentCenterStateReason reason) = 0;

    /**
     * Simple info callback of getSongSimpleInfo, occurs when getSongSimpleInfo is called
     * 
     * @param requestId The request id is same as that returned by getSongSimpleInfo.
     * @param songCode Song code
     * @param simpleInfo The metadata of the music.
     * @param reason The status of the request. See MusicContentCenterStateReason
     */
    virtual void onSongSimpleInfoResult(const char* requestId, int64_t songCode, const char* simpleInfo, MusicContentCenterStateReason reason) = 0;

    /**
     * Preload process callback, occurs when preload is called
     * 
     * @param requestId The request id is same as that returned by preload.
     * @param songCode Song code
     * @param percent Preload progress (0 ~ 100)
     * @param lyricUrl  The lyric url of this music
     * @param state Preload state; see PreloadState.
     * @param reason The status of the request. See MusicContentCenterStateReason
     */
    virtual void onPreLoadEvent(const char* requestId, int64_t internalSongCode, int percent, const char* payload, MusicContentCenterState status, MusicContentCenterStateReason reason) = 0;
    
    /**
     * Score result callback, occurs when startScore is called
     * 
     * @param songCode Song code
     * @param status The status of the request. See MusicContentCenterState
     * @param reason The status of the request. See MusicContentCenterStateReason
     */
    virtual void onStartScoreResult(int64_t internalSongCode,  MusicContentCenterState status, MusicContentCenterStateReason reason) = 0;

    virtual ~IMusicContentCenterEventHandler() {};
};

struct MusicContentCenterConfiguration {
    /**
     * The maximum number of cached media files.
     */
    int32_t maxCacheSize;
    /**
     * Event handler to get callback result.
     */
    IMusicContentCenterEventHandler* eventHandler;
    /**
     * Event handler to get callback result.
     */
    IScoreEventHandler* scoreEventHandler;
    /**
     * Audio frame observer to get audio raw data.
     */
    media::IAudioFrameObserver* audioFrameObserver;

    MusicContentCenterConfiguration():eventHandler(nullptr),scoreEventHandler(nullptr),maxCacheSize(20), audioFrameObserver(nullptr){}
    MusicContentCenterConfiguration(int32_t maxCacheSize, IMusicContentCenterEventHandler* eventHandler, IScoreEventHandler* scoreEventHandler,  media::IAudioFrameObserver* audioFrameObserver) : maxCacheSize(maxCacheSize), eventHandler(eventHandler), scoreEventHandler(scoreEventHandler), audioFrameObserver(audioFrameObserver) {}
};

struct MusicContentCenterVendorDefaultConfiguration {
    /**
     * The app ID of the project that has enabled the music content center
     */
    const char* appId;
    /**
     * Music content center need token to connect with server
     */
    const char* token;
    /**
     * The user ID when using music content center. It can be different from that of the rtc product.
     */
    const char* userId;
    /**
     * @technical preview
     */
    const char* mccDomain;
};

struct MusicContentCenterVendor2Configuration {
    /**
     * The app ID of the vendor2 that has enabled the music content center
     */
    const char* appId;
    /**
     * The app key of the vendor2 that has enabled the music content center
     */
    const char* appKey;
    /**
     * The token of the vendor2 that has enabled the music content center
     */
    const char* token;
    /**
     * The user ID of the vendor2 that has enabled the music content center
     */
    const char* userId;
    /**
     * The device ID of the vendor2 that has enabled the music content center
     */
    const char* deviceId;
    /**
     * The url token expire time of the vendor2 that has enabled the music content center
     */
    int32_t urlTokenExpireTime;
    /**
     * The charge mode of the vendor2 that has enabled the music content center
     */
    int32_t chargeMode;
};

class IMusicPlayer : public IMediaPlayer {
protected:
    virtual ~IMusicPlayer() {};

public:
    IMusicPlayer() {};
    using IMediaPlayer::open;
    /**
    * Open a media file with specified parameters.
    *
    * @param songCode The identifier of the media file that you want to play.
    * @param startPos The playback position (ms) of the music file.
    * @return
    * - 0: Success.
    * - < 0: Failure.
    */
    virtual int open(int64_t songCode, int64_t startPos = 0) = 0;

    /**
    * Set the mode for playing songs.
    * You can call this method to switch from original to accompaniment or lead vocals.
    * If you do not call this method to set the mode, the SDK plays the accompaniment by default.
    *
    * @param model The playing mode.
    * @return
    * - 0: Success.
    * - < 0: Failure.
    */
    virtual int setPlayMode(MusicPlayMode mode) = 0;
};

class IMusicContentCenter : public RefCountInterface
{
protected:
    virtual ~IMusicContentCenter(){};
public:
    IMusicContentCenter() {};

    /**
     * Initializes the IMusicContentCenter
     * Set token of music content center and other params
     *
     * @param configuration
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int initialize(const MusicContentCenterConfiguration & configuration) = 0;

    /**
     * Add vendor to music content center
     *
     * @param vendorId The vendor id.
     * @param jsonVendorConfig The json string of vendor config.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int addVendor(MccVendorID vendorId, const char* jsonVendorConfig) = 0;

    /**
     * Remove vendor from music content center
     *
     * @param vendorId The vendor id.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int removeVendor(MccVendorID vendorId) = 0;

    /**
     * Renew token of music content center
     * 
     * @param vendorID The vendor id.
     * @param token The new token.
     * @return 
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int renewToken(MccVendorID vendorID, const char* token) = 0;

    /**
     * release music content center resource.
     * 
     */
    virtual void release() = 0;

    /**
    *  register event handler.
    */
    virtual int registerEventHandler(IMusicContentCenterEventHandler* eventHandler) = 0;
    
    /**
    *  unregister event handler.
    */
    virtual int unregisterEventHandler(IMusicContentCenterEventHandler* eventHandler) = 0;

    /**
     * Creates a music player source object and return its pointer.
     * @return
     * - The pointer to \ref rtc::IMusicPlayer "IMusicPlayer",
     *   if the method call succeeds.
     * - The empty pointer NULL, if the method call fails.
     */
    virtual agora_refptr<IMusicPlayer> createMusicPlayer() = 0;

    /**
     * Destroy a music player source object and return result.
     * @param music_player The pointer to \ref rtc::IMusicPlayer "IMusicPlayer".
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int destroyMusicPlayer(agora_refptr<IMusicPlayer> music_player) = 0;
    
    /**
     * Get music chart collection of music.
     * If the method call succeeds, get result from the 
     * \ref agora::rtc::IMusicContentCenterEventHandler::onMusicChartsResult
     * "onMusicChartsResult" callback 
     * @param requestId The request id you will get of this query, format is uuid.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int getMusicCharts(agora::util::AString& requestId) = 0;
    
    /**
     * Get music collection of the music chart by musicChartId and page info. 
     * If the method call success, get result from the 
     * \ref agora::rtc::IMusicContentCenterEventHandler::onMusicCollectionResult
     * "onMusicCollectionResult" callback 
     * @param requestId The request id you will get of this query, format is uuid.
     * @param musicChartId The music chart id obtained from getMusicCharts.
     * @param page The page of the music chart, starting from 1
     * @param pageSize The page size, max is 50.
     * @param jsonOption The ext param, default is null.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int getMusicCollectionByMusicChartId(agora::util::AString& requestId, int32_t musicChartId, int32_t page, int32_t pageSize, const char* jsonOption = nullptr) = 0;
    
    /**
     * Search music by keyword and page info.
     * If the method call success, get result from the 
     * \ref agora::rtc::IMusicContentCenterEventHandler::onMusicCollectionResult
     * "onMusicCollectionResult" callback 
     * @param requestId The request id you will get of this query, format is uuid.
     * @param keyWord The key word to search.
     * @param page The page of music search result , start from 1.
     * @param pageSize The page size, max is 50.
     * @param jsonOption The ext param, default is null.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int searchMusic(agora::util::AString& requestId, const char* keyWord, int32_t page, int32_t pageSize, const char* jsonOption = nullptr) = 0;

    /**
     * Preload a media file with specified parameters.
     * 
     * @param requestId The request id you will get of this query, format is uuid.
     * @param songCode The identifier of the media file that you want to play.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int preload(agora::util::AString& requestId, int64_t internalSongCode) = 0;

     /**
    *  register score event handler.
    */
    virtual int registerScoreEventHandler(IScoreEventHandler* scoreEventHandler) = 0;
    
    /**
    *  unregister score event handler.
    */
    virtual int unregisterScoreEventHandler(IScoreEventHandler* scoreEventHandler) = 0;

    /**
     * Set the score level of the music player.
     * 
     * @param level The score level of the music.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int setScoreLevel(ScoreLevel level) = 0;

    /**
     * Start score of the songCode.
     * 
     * @param internalSongCode The identifier of the media file that you play.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int startScore(int64_t internalSongCode) = 0;


    /**
     * Stop score.
     * 
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int stopScore() = 0;


    /**
     * Pause score of the songCode.
     * 
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int pauseScore() = 0;

    /**
     * Resume score of the songCode.
     * 
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int resumeScore() = 0;

    /**
     * Get cumulative score data of the songCode.
     * 
     * @param cumulativeScoreData The cumulative score data of the songCode, you will get the result from this parameter.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int getCumulativeScoreData(CumulativeScoreData &cumulativeScoreData) = 0;

    /**
     * Remove a media file cache
     *
     * @param internalSongCode The identifier of the media file that you want to play.
     * @return
     * - 0: Success; the cached media file is removed.
     * - < 0: Failure.
     */
    virtual int removeCache(int64_t internalSongCode) = 0;

    /**
     * Get cached media files.
     * Before calling this API, you should allocate a memory buffer that stores the cached media file information, and pass the pointer of the buffer as the input parameter cacheInfo, and set the size of the memory buffer to cacheInfoSize.
     * The sample code below illustrates how to request the cached media file information:
     *
     *    cacheInfoSize = 10                                                                    // Allocate a memory buffer of 10 MusicCacheInfo size
     *    agora::rtc::MusicCacheInfo *infos = new agora::rtc::MusicCacheInfo[cacheInfoSize];
     *    int ret = self.imcc->getCaches(infos, cacheInfoSize);
     *    if (ret < 0) {                                                                        // error occurred!
     *        return; 
     *    }
     *    std::cout << "the cache size:" << cacheInfoSize << std::endl;                         // The cache size: 5
     *
     *
     * @param cacheInfo An output parameter; A pointer to the memory buffer that stores the cached media file information. The memory buffer pointed to by cacheInfo should be allocated by yourself before calling this API.
     * @param cacheInfoSize
     * - Input: The number of MusicCacheInfo's size that you get from the memory. 
     * - Output: The actual number of MusicCacheInfo struct that is returned.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int getCaches(MusicCacheInfo *cacheInfo, int32_t* cacheInfoSize) = 0;

    /**
     * Check if the media file is preloaded
     *
     * @param internalSongCode The identifier of the media file that you want to play.
     * @return
     * - 0: Success, file is preloaded.
     * - < 0: Failure.
     */
    virtual int isPreloaded(int64_t internalSongCode) = 0;

    /**
     * Get lyric of the music.
     *
     * @param requestId The request id you will get of this query, format is uuid.
     * @param internalSongCode The identifier of the media file that you want to play.
     * @param lyricType The type of the lyric file. 0:xml or 1:lrc.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int getLyric(agora::util::AString& requestId, int64_t internalSongCode, int32_t lyricType = 0) = 0;

    /**
     * Get lyric info of the music.
     *
     * @param requestId The request id you will get of this query, format is uuid.
     * @param internalSongCode The identifier of the media file that you want to play.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int getLyricInfo(agora::util::AString& requestId, int64_t internalSongCode) = 0;
    /**
     * Gets the metadata of a specific music. Once this method is called, the SDK triggers the onSongSimpleInfoResult callback to report the metadata of the music.
     *
     * @param requestId The request id you will get of this query, format is uuid.
     * @param internalSongCode The identifier of the media file.
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int getSongSimpleInfo(agora::util::AString& requestId, int64_t internalSongCode) = 0;

    /**
     * Get internal songCodeKey from songCode and jsonOption
     * 
     * @param vendorId The vendor id.
     * @param songCode The identifier of the media file.
     * @param jsonOption An extention parameter. The default value is null. it’s a json-format string and the `key` and `value` can be customized according to your scenarios.
     * @param internalSongCode The identifier of internal 
     * @return
     * - 0: Success.
     * - < 0: Failure.
     */
    virtual int getInternalSongCode(MccVendorID vendorId, const char* songCode, const char* jsonOption,
                                  int64_t& internalSongCode) = 0;
};

}  // namespace rtc
}  // namespace agora
