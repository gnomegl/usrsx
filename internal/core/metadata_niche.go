package core

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

func extractTinderMetadata(responseText string, responseCode int) *ProfileMetadata {
	metadata := &ProfileMetadata{
		CustomFields:    make(map[string]string),
		AdditionalLinks: make(map[string]string),
	}

	nameRe := regexp.MustCompile(`<meta\s+property=["']og:title["']\s+content=["']([^"']+)["']`)
	if matches := nameRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.DisplayName = matches[1]
	}

	descRe := regexp.MustCompile(`<meta\s+property=["']og:description["']\s+content=["']([^"']+)["']`)
	if matches := descRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.Bio = matches[1]
	}

	avatarRe := regexp.MustCompile(`<meta\s+property=["']og:image["']\s+content=["']([^"']+)["']`)
	if matches := avatarRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.AvatarURL = matches[1]
	}

	photoCountRe := regexp.MustCompile(`"photos":\s*\[([^\]]+)\]`)
	if matches := photoCountRe.FindStringSubmatch(responseText); len(matches) > 1 {
		photosStr := matches[1]
		photoCount := strings.Count(photosStr, "url")
		if photoCount > 0 {
			metadata.CustomFields["photos"] = strconv.Itoa(photoCount)
		}
	}

	return metadata
}

func extractNewgroundsMetadataNew(responseText string, responseCode int) *ProfileMetadata {
	metadata := &ProfileMetadata{
		CustomFields:    make(map[string]string),
		AdditionalLinks: make(map[string]string),
	}

	nameRe := regexp.MustCompile(`<meta\s+property=["']og:title["']\s+content=["']([^"']+)["']`)
	if matches := nameRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.DisplayName = matches[1]
	}

	descRe := regexp.MustCompile(`<meta\s+property=["']og:description["']\s+content=["']([^"']+)["']`)
	if matches := descRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.Bio = matches[1]
	}

	avatarRe := regexp.MustCompile(`<meta\s+property=["']og:image["']\s+content=["']([^"']+)["']`)
	if matches := avatarRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.AvatarURL = matches[1]
	}

	fansRe := regexp.MustCompile(`([0-9,]+)\s+Fans`)
	if matches := fansRe.FindStringSubmatch(responseText); len(matches) > 1 {
		countStr := strings.ReplaceAll(matches[1], ",", "")
		if count, err := strconv.Atoi(countStr); err == nil {
			metadata.FollowerCount = count
			metadata.CustomFields["fans"] = matches[1]
		}
	}

	submissionsRe := regexp.MustCompile(`([0-9,]+)\s+(?:Art|Submissions)`)
	if matches := submissionsRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.CustomFields["submissions"] = matches[1]
	}

	return metadata
}

func extractCohostMetadataNew(responseText string, responseCode int) *ProfileMetadata {
	metadata := &ProfileMetadata{
		CustomFields:    make(map[string]string),
		AdditionalLinks: make(map[string]string),
	}

	nameRe := regexp.MustCompile(`<meta\s+property=["']og:title["']\s+content=["']([^"']+)["']`)
	if matches := nameRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.DisplayName = matches[1]
	}

	descRe := regexp.MustCompile(`<meta\s+property=["']og:description["']\s+content=["']([^"']+)["']`)
	if matches := descRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.Bio = matches[1]
	}

	avatarRe := regexp.MustCompile(`<meta\s+property=["']og:image["']\s+content=["']([^"']+)["']`)
	if matches := avatarRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.AvatarURL = matches[1]
	}

	return metadata
}

func extractLemmyMetadataNew(responseText string, responseCode int) *ProfileMetadata {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &data); err != nil {
		return extractGenericHTMLMetadata(responseText)
	}

	metadata := &ProfileMetadata{
		CustomFields:    make(map[string]string),
		AdditionalLinks: make(map[string]string),
	}

	if personView, ok := data["person_view"].(map[string]interface{}); ok {
		if person, ok := personView["person"].(map[string]interface{}); ok {
			if name, ok := person["name"].(string); ok {
				metadata.CustomFields["username"] = name
			}
			if displayName, ok := person["display_name"].(string); ok {
				metadata.DisplayName = displayName
			}
			if bio, ok := person["bio"].(string); ok {
				metadata.Bio = bio
			}
			if avatar, ok := person["avatar"].(string); ok {
				metadata.AvatarURL = avatar
			}
			if banner, ok := person["banner"].(string); ok {
				metadata.CustomFields["banner"] = banner
			}
			if published, ok := person["published"].(string); ok {
				metadata.JoinDate = published
			}
		}
		if counts, ok := personView["counts"].(map[string]interface{}); ok {
			if postCount, ok := counts["post_count"].(float64); ok {
				metadata.CustomFields["posts"] = strconv.Itoa(int(postCount))
			}
			if commentCount, ok := counts["comment_count"].(float64); ok {
				metadata.CustomFields["comments"] = strconv.Itoa(int(commentCount))
			}
		}
	}

	return metadata
}

func extractBlueskyMetadataNew(responseText string, responseCode int) *ProfileMetadata {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &data); err != nil {
		return extractGenericHTMLMetadata(responseText)
	}

	metadata := &ProfileMetadata{
		CustomFields:    make(map[string]string),
		AdditionalLinks: make(map[string]string),
	}

	if displayName, ok := data["displayName"].(string); ok {
		metadata.DisplayName = displayName
	}
	if handle, ok := data["handle"].(string); ok {
		metadata.CustomFields["handle"] = handle
	}
	if description, ok := data["description"].(string); ok {
		metadata.Bio = description
	}
	if avatar, ok := data["avatar"].(string); ok {
		metadata.AvatarURL = avatar
	}
	if followersCount, ok := data["followersCount"].(float64); ok {
		metadata.FollowerCount = int(followersCount)
	}
	if followsCount, ok := data["followsCount"].(float64); ok {
		metadata.FollowingCount = int(followsCount)
	}
	if postsCount, ok := data["postsCount"].(float64); ok {
		metadata.CustomFields["posts"] = strconv.Itoa(int(postsCount))
	}
	if indexedAt, ok := data["indexedAt"].(string); ok {
		metadata.JoinDate = indexedAt
	}

	return metadata
}

func extractThreadsMetadataNew(responseText string, responseCode int) *ProfileMetadata {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &data); err != nil {
		return extractGenericHTMLMetadata(responseText)
	}

	metadata := &ProfileMetadata{
		CustomFields:    make(map[string]string),
		AdditionalLinks: make(map[string]string),
	}

	if user, ok := data["user"].(map[string]interface{}); ok {
		if username, ok := user["username"].(string); ok {
			metadata.CustomFields["username"] = username
		}
		if fullName, ok := user["full_name"].(string); ok {
			metadata.DisplayName = fullName
		}
		if biography, ok := user["biography"].(string); ok {
			metadata.Bio = biography
		}
		if profilePicURL, ok := user["profile_pic_url"].(string); ok {
			metadata.AvatarURL = profilePicURL
		}
		if followerCount, ok := user["follower_count"].(float64); ok {
			metadata.FollowerCount = int(followerCount)
		}
	}

	metadata2 := extractGenericHTMLMetadata(responseText)
	if metadata.DisplayName == "" && metadata2.DisplayName != "" {
		metadata.DisplayName = metadata2.DisplayName
	}
	if metadata.Bio == "" && metadata2.Bio != "" {
		metadata.Bio = metadata2.Bio
	}
	if metadata.AvatarURL == "" && metadata2.AvatarURL != "" {
		metadata.AvatarURL = metadata2.AvatarURL
	}

	return metadata
}

func extractPolyworkMetadataNew(responseText string, responseCode int) *ProfileMetadata {
	metadata := &ProfileMetadata{
		CustomFields:    make(map[string]string),
		AdditionalLinks: make(map[string]string),
	}

	nameRe := regexp.MustCompile(`<meta\s+property=["']og:title["']\s+content=["']([^"']+)["']`)
	if matches := nameRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.DisplayName = matches[1]
	}

	descRe := regexp.MustCompile(`<meta\s+property=["']og:description["']\s+content=["']([^"']+)["']`)
	if matches := descRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.Bio = matches[1]
	}

	avatarRe := regexp.MustCompile(`<meta\s+property=["']og:image["']\s+content=["']([^"']+)["']`)
	if matches := avatarRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.AvatarURL = matches[1]
	}

	headlineRe := regexp.MustCompile(`"headline"\s*:\s*"([^"]+)"`)
	if matches := headlineRe.FindStringSubmatch(responseText); len(matches) > 1 && metadata.Bio == "" {
		metadata.CustomFields["headline"] = matches[1]
	}

	locationRe := regexp.MustCompile(`"location"\s*:\s*"([^"]+)"`)
	if matches := locationRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.Location = matches[1]
	}

	return metadata
}

func extractContraMetadataNew(responseText string, responseCode int) *ProfileMetadata {
	metadata := &ProfileMetadata{
		CustomFields:    make(map[string]string),
		AdditionalLinks: make(map[string]string),
	}

	nameRe := regexp.MustCompile(`<meta\s+property=["']og:title["']\s+content=["']([^"']+)["']`)
	if matches := nameRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.DisplayName = matches[1]
	}

	descRe := regexp.MustCompile(`<meta\s+property=["']og:description["']\s+content=["']([^"']+)["']`)
	if matches := descRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.Bio = matches[1]
	}

	avatarRe := regexp.MustCompile(`<meta\s+property=["']og:image["']\s+content=["']([^"']+)["']`)
	if matches := avatarRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.AvatarURL = matches[1]
	}

	skillsRe := regexp.MustCompile(`"skills"\s*:\s*\[([^\]]+)\]`)
	if matches := skillsRe.FindStringSubmatch(responseText); len(matches) > 1 {
		skillsStr := strings.ReplaceAll(matches[1], `"`, "")
		metadata.CustomFields["skills"] = skillsStr
	}

	portfolioCountRe := regexp.MustCompile(`([0-9]+)\s+(?:projects?|portfolio items?)`)
	if matches := portfolioCountRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.CustomFields["portfolio_items"] = matches[1]
	}

	return metadata
}

func extractAboutMeMetadataNew(responseText string, responseCode int) *ProfileMetadata {
	metadata := &ProfileMetadata{
		CustomFields:    make(map[string]string),
		AdditionalLinks: make(map[string]string),
	}

	nameRe := regexp.MustCompile(`<meta\s+property=["']og:title["']\s+content=["']([^"']+)["']`)
	if matches := nameRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.DisplayName = matches[1]
	}

	descRe := regexp.MustCompile(`<meta\s+property=["']og:description["']\s+content=["']([^"']+)["']`)
	if matches := descRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.Bio = matches[1]
	}

	avatarRe := regexp.MustCompile(`<meta\s+property=["']og:image["']\s+content=["']([^"']+)["']`)
	if matches := avatarRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.AvatarURL = matches[1]
	}

	linksRe := regexp.MustCompile(`<a[^>]+href=["']([^"']+)["'][^>]*class=["'][^"']*(?:link|social)[^"']*["']`)
	links := linksRe.FindAllStringSubmatch(responseText, -1)
	linkCount := 0
	for _, match := range links {
		if len(match) > 1 {
			linkCount++
			metadata.AdditionalLinks["link_"+strconv.Itoa(linkCount)] = match[1]
		}
	}
	if linkCount > 0 {
		metadata.CustomFields["link_count"] = strconv.Itoa(linkCount)
	}

	return metadata
}

func extractProductHuntMetadataNew(responseText string, responseCode int) *ProfileMetadata {
	metadata := &ProfileMetadata{
		CustomFields:    make(map[string]string),
		AdditionalLinks: make(map[string]string),
	}

	nameRe := regexp.MustCompile(`<meta\s+property=["']og:title["']\s+content=["']([^"']+)["']`)
	if matches := nameRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.DisplayName = matches[1]
	}

	descRe := regexp.MustCompile(`<meta\s+property=["']og:description["']\s+content=["']([^"']+)["']`)
	if matches := descRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.Bio = matches[1]
	}

	avatarRe := regexp.MustCompile(`<meta\s+property=["']og:image["']\s+content=["']([^"']+)["']`)
	if matches := avatarRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.AvatarURL = matches[1]
	}

	upvotesRe := regexp.MustCompile(`([0-9,]+)\s+(?:upvotes?|points?)`)
	if matches := upvotesRe.FindStringSubmatch(responseText); len(matches) > 1 {
		countStr := strings.ReplaceAll(matches[1], ",", "")
		if count, err := strconv.Atoi(countStr); err == nil {
			metadata.CustomFields["upvotes"] = strconv.Itoa(count)
		}
	}

	productsRe := regexp.MustCompile(`([0-9,]+)\s+products?`)
	if matches := productsRe.FindStringSubmatch(responseText); len(matches) > 1 {
		metadata.CustomFields["products"] = matches[1]
	}

	followersRe := regexp.MustCompile(`([0-9,]+)\s+followers?`)
	if matches := followersRe.FindStringSubmatch(responseText); len(matches) > 1 {
		countStr := strings.ReplaceAll(matches[1], ",", "")
		if count, err := strconv.Atoi(countStr); err == nil {
			metadata.FollowerCount = count
		}
	}

	return metadata
}
