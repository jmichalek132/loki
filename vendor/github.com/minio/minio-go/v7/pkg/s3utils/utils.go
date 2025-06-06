/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2020 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package s3utils

import (
	"bytes"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Sentinel URL is the default url value which is invalid.
var sentinelURL = url.URL{}

// IsValidDomain validates if input string is a valid domain name.
func IsValidDomain(host string) bool {
	// See RFC 1035, RFC 3696.
	host = strings.TrimSpace(host)
	if len(host) == 0 || len(host) > 255 {
		return false
	}
	// host cannot start or end with "-"
	if host[len(host)-1:] == "-" || host[:1] == "-" {
		return false
	}
	// host cannot start or end with "_"
	if host[len(host)-1:] == "_" || host[:1] == "_" {
		return false
	}
	// host cannot start with a "."
	if host[:1] == "." {
		return false
	}
	// All non alphanumeric characters are invalid.
	if strings.ContainsAny(host, "`~!@#$%^&*()+={}[]|\\\"';:><?/") {
		return false
	}
	// No need to regexp match, since the list is non-exhaustive.
	// We let it valid and fail later.
	return true
}

// IsValidIP parses input string for ip address validity.
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsVirtualHostSupported - verifies if bucketName can be part of
// virtual host. Currently only Amazon S3 and Google Cloud Storage
// would support this.
func IsVirtualHostSupported(endpointURL url.URL, bucketName string) bool {
	if endpointURL == sentinelURL {
		return false
	}
	// bucketName can be valid but '.' in the hostname will fail SSL
	// certificate validation. So do not use host-style for such buckets.
	if endpointURL.Scheme == "https" && strings.Contains(bucketName, ".") {
		return false
	}
	// Return true for all other cases
	return IsAmazonEndpoint(endpointURL) || IsGoogleEndpoint(endpointURL) || IsAliyunOSSEndpoint(endpointURL)
}

// Refer for region styles - https://docs.aws.amazon.com/general/latest/gr/rande.html#s3_region

// amazonS3HostHyphen - regular expression used to determine if an arg is s3 host in hyphenated style.
var amazonS3HostHyphen = regexp.MustCompile(`^s3-(.*?).amazonaws.com$`)

// amazonS3HostDualStack - regular expression used to determine if an arg is s3 host dualstack.
var amazonS3HostDualStack = regexp.MustCompile(`^s3.dualstack.(.*?).amazonaws.com$`)

// amazonS3HostFIPS - regular expression used to determine if an arg is s3 FIPS host.
var amazonS3HostFIPS = regexp.MustCompile(`^s3-fips.(.*?).amazonaws.com$`)

// amazonS3HostFIPSDualStack - regular expression used to determine if an arg is s3 FIPS host dualstack.
var amazonS3HostFIPSDualStack = regexp.MustCompile(`^s3-fips.dualstack.(.*?).amazonaws.com$`)

// amazonS3HostExpress - regular expression used to determine if an arg is S3 Express zonal endpoint.
var amazonS3HostExpress = regexp.MustCompile(`^s3express-[a-z0-9]{3,7}-az[1-6]\.([a-z0-9-]+)\.amazonaws\.com$`)

// amazonS3HostExpressControl - regular expression used to determine if an arg is S3 express regional endpoint.
var amazonS3HostExpressControl = regexp.MustCompile(`^s3express-control\.([a-z0-9-]+)\.amazonaws\.com$`)

// amazonS3HostDot - regular expression used to determine if an arg is s3 host in . style.
var amazonS3HostDot = regexp.MustCompile(`^s3.(.*?).amazonaws.com$`)

// amazonS3ChinaHost - regular expression used to determine if the arg is s3 china host.
var amazonS3ChinaHost = regexp.MustCompile(`^s3.(cn.*?).amazonaws.com.cn$`)

// amazonS3ChinaHostDualStack - regular expression used to determine if the arg is s3 china host dualstack.
var amazonS3ChinaHostDualStack = regexp.MustCompile(`^s3.dualstack.(cn.*?).amazonaws.com.cn$`)

// Regular expression used to determine if the arg is elb host.
var elbAmazonRegex = regexp.MustCompile(`elb(.*?).amazonaws.com$`)

// Regular expression used to determine if the arg is elb host in china.
var elbAmazonCnRegex = regexp.MustCompile(`elb(.*?).amazonaws.com.cn$`)

// amazonS3HostPrivateLink - regular expression used to determine if an arg is s3 host in AWS PrivateLink interface endpoints style
var amazonS3HostPrivateLink = regexp.MustCompile(`^(?:bucket|accesspoint).vpce-.*?.s3.(.*?).vpce.amazonaws.com$`)

// GetRegionFromURL - returns a region from url host.
func GetRegionFromURL(endpointURL url.URL) string {
	if endpointURL == sentinelURL {
		return ""
	}

	if endpointURL.Hostname() == "s3-external-1.amazonaws.com" {
		return ""
	}

	// if elb's are used we cannot calculate which region it may be, just return empty.
	if elbAmazonRegex.MatchString(endpointURL.Hostname()) || elbAmazonCnRegex.MatchString(endpointURL.Hostname()) {
		return ""
	}

	// We check for FIPS dualstack matching first to avoid the non-greedy
	// regex for FIPS non-dualstack matching a dualstack URL
	parts := amazonS3HostFIPSDualStack.FindStringSubmatch(endpointURL.Hostname())
	if len(parts) > 1 {
		return parts[1]
	}

	parts = amazonS3HostFIPS.FindStringSubmatch(endpointURL.Hostname())
	if len(parts) > 1 {
		return parts[1]
	}

	parts = amazonS3HostDualStack.FindStringSubmatch(endpointURL.Hostname())
	if len(parts) > 1 {
		return parts[1]
	}

	parts = amazonS3HostHyphen.FindStringSubmatch(endpointURL.Hostname())
	if len(parts) > 1 {
		return parts[1]
	}

	parts = amazonS3ChinaHost.FindStringSubmatch(endpointURL.Hostname())
	if len(parts) > 1 {
		return parts[1]
	}

	parts = amazonS3ChinaHostDualStack.FindStringSubmatch(endpointURL.Hostname())
	if len(parts) > 1 {
		return parts[1]
	}

	parts = amazonS3HostPrivateLink.FindStringSubmatch(endpointURL.Hostname())
	if len(parts) > 1 {
		return parts[1]
	}

	parts = amazonS3HostExpress.FindStringSubmatch(endpointURL.Hostname())
	if len(parts) > 1 {
		return parts[1]
	}

	parts = amazonS3HostExpressControl.FindStringSubmatch(endpointURL.Hostname())
	if len(parts) > 1 {
		return parts[1]
	}

	parts = amazonS3HostDot.FindStringSubmatch(endpointURL.Hostname())
	if len(parts) > 1 {
		if strings.HasPrefix(parts[1], "xpress-") {
			return ""
		}
		if strings.HasPrefix(parts[1], "dualstack.") || strings.HasPrefix(parts[1], "control.") || strings.HasPrefix(parts[1], "website-") {
			return ""
		}
		return parts[1]
	}

	return ""
}

// IsAliyunOSSEndpoint - Match if it is exactly Aliyun OSS endpoint.
func IsAliyunOSSEndpoint(endpointURL url.URL) bool {
	return strings.HasSuffix(endpointURL.Hostname(), "aliyuncs.com")
}

// IsAmazonExpressRegionalEndpoint Match if the endpoint is S3 Express regional endpoint.
func IsAmazonExpressRegionalEndpoint(endpointURL url.URL) bool {
	return amazonS3HostExpressControl.MatchString(endpointURL.Hostname())
}

// IsAmazonExpressZonalEndpoint Match if the endpoint is S3 Express zonal endpoint.
func IsAmazonExpressZonalEndpoint(endpointURL url.URL) bool {
	return amazonS3HostExpress.MatchString(endpointURL.Hostname())
}

// IsAmazonEndpoint - Match if it is exactly Amazon S3 endpoint.
func IsAmazonEndpoint(endpointURL url.URL) bool {
	if endpointURL.Hostname() == "s3-external-1.amazonaws.com" || endpointURL.Hostname() == "s3.amazonaws.com" {
		return true
	}
	return GetRegionFromURL(endpointURL) != ""
}

// IsAmazonGovCloudEndpoint - Match if it is exactly Amazon S3 GovCloud endpoint.
func IsAmazonGovCloudEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	return (endpointURL.Host == "s3-us-gov-west-1.amazonaws.com" ||
		endpointURL.Host == "s3-us-gov-east-1.amazonaws.com" ||
		IsAmazonFIPSGovCloudEndpoint(endpointURL))
}

// IsAmazonFIPSGovCloudEndpoint - match if the endpoint is FIPS and GovCloud.
func IsAmazonFIPSGovCloudEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	return IsAmazonFIPSEndpoint(endpointURL) && strings.Contains(endpointURL.Hostname(), "us-gov-")
}

// IsAmazonFIPSEndpoint - Match if it is exactly Amazon S3 FIPS endpoint.
// See https://aws.amazon.com/compliance/fips.
func IsAmazonFIPSEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	return strings.HasPrefix(endpointURL.Hostname(), "s3-fips") && strings.HasSuffix(endpointURL.Hostname(), ".amazonaws.com")
}

// IsAmazonPrivateLinkEndpoint - Match if it is exactly Amazon S3 PrivateLink interface endpoint
// See https://docs.aws.amazon.com/AmazonS3/latest/userguide/privatelink-interface-endpoints.html.
func IsAmazonPrivateLinkEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	return amazonS3HostPrivateLink.MatchString(endpointURL.Hostname())
}

// IsGoogleEndpoint - Match if it is exactly Google cloud storage endpoint.
func IsGoogleEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	return endpointURL.Hostname() == "storage.googleapis.com"
}

// Expects ascii encoded strings - from output of urlEncodePath
func percentEncodeSlash(s string) string {
	return strings.ReplaceAll(s, "/", "%2F")
}

// QueryEncode - encodes query values in their URL encoded form. In
// addition to the percent encoding performed by urlEncodePath() used
// here, it also percent encodes '/' (forward slash)
func QueryEncode(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		prefix := percentEncodeSlash(EncodePath(k)) + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(prefix)
			buf.WriteString(percentEncodeSlash(EncodePath(v)))
		}
	}
	return buf.String()
}

// if object matches reserved string, no need to encode them
var reservedObjectNames = regexp.MustCompile("^[a-zA-Z0-9-_.~/]+$")

// EncodePath encode the strings from UTF-8 byte representations to HTML hex escape sequences
//
// This is necessary since regular url.Parse() and url.Encode() functions do not support UTF-8
// non english characters cannot be parsed due to the nature in which url.Encode() is written
//
// This function on the other hand is a direct replacement for url.Encode() technique to support
// pretty much every UTF-8 character.
func EncodePath(pathName string) string {
	if reservedObjectNames.MatchString(pathName) {
		return pathName
	}
	var encodedPathname strings.Builder
	for _, s := range pathName {
		if 'A' <= s && s <= 'Z' || 'a' <= s && s <= 'z' || '0' <= s && s <= '9' { // §2.3 Unreserved characters (mark)
			encodedPathname.WriteRune(s)
			continue
		}
		switch s {
		case '-', '_', '.', '~', '/': // §2.3 Unreserved characters (mark)
			encodedPathname.WriteRune(s)
			continue
		default:
			l := utf8.RuneLen(s)
			if l < 0 {
				// if utf8 cannot convert return the same string as is
				return pathName
			}
			u := make([]byte, l)
			utf8.EncodeRune(u, s)
			for _, r := range u {
				hex := hex.EncodeToString([]byte{r})
				encodedPathname.WriteString("%" + strings.ToUpper(hex))
			}
		}
	}
	return encodedPathname.String()
}

// We support '.' with bucket names but we fallback to using path
// style requests instead for such buckets.
var (
	validBucketName          = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9\.\-\_\:]{1,61}[A-Za-z0-9]$`)
	validBucketNameStrict    = regexp.MustCompile(`^[a-z0-9][a-z0-9\.\-]{1,61}[a-z0-9]$`)
	validBucketNameS3Express = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]--[a-z0-9]{3,7}-az[1-6]--x-s3$`)
	ipAddress                = regexp.MustCompile(`^(\d+\.){3}\d+$`)
)

// Common checker for both stricter and basic validation.
func checkBucketNameCommon(bucketName string, strict bool) (err error) {
	if strings.TrimSpace(bucketName) == "" {
		return errors.New("Bucket name cannot be empty")
	}
	if len(bucketName) < 3 {
		return errors.New("Bucket name cannot be shorter than 3 characters")
	}
	if len(bucketName) > 63 {
		return errors.New("Bucket name cannot be longer than 63 characters")
	}
	if ipAddress.MatchString(bucketName) {
		return errors.New("Bucket name cannot be an ip address")
	}
	if strings.Contains(bucketName, "..") || strings.Contains(bucketName, ".-") || strings.Contains(bucketName, "-.") {
		return errors.New("Bucket name contains invalid characters")
	}
	if strict {
		if !validBucketNameStrict.MatchString(bucketName) {
			err = errors.New("Bucket name contains invalid characters")
		}
		return err
	}
	if !validBucketName.MatchString(bucketName) {
		err = errors.New("Bucket name contains invalid characters")
	}
	return err
}

// CheckValidBucketName - checks if we have a valid input bucket name.
func CheckValidBucketName(bucketName string) (err error) {
	return checkBucketNameCommon(bucketName, false)
}

// IsS3ExpressBucket is S3 express bucket?
func IsS3ExpressBucket(bucketName string) bool {
	return CheckValidBucketNameS3Express(bucketName) == nil
}

// CheckValidBucketNameS3Express - checks if we have a valid input bucket name for S3 Express.
func CheckValidBucketNameS3Express(bucketName string) (err error) {
	if strings.TrimSpace(bucketName) == "" {
		return errors.New("Bucket name cannot be empty for S3 Express")
	}

	if len(bucketName) < 3 {
		return errors.New("Bucket name cannot be shorter than 3 characters for S3 Express")
	}

	if len(bucketName) > 63 {
		return errors.New("Bucket name cannot be longer than 63 characters for S3 Express")
	}

	// Check if the bucket matches the regex
	if !validBucketNameS3Express.MatchString(bucketName) {
		return errors.New("Bucket name contains invalid characters")
	}

	// Extract bucket name (before --<az-id>--x-s3)
	parts := strings.Split(bucketName, "--")
	if len(parts) != 3 || parts[2] != "x-s3" {
		return errors.New("Bucket name pattern is wrong 'x-s3'")
	}
	bucketName = parts[0]

	// Additional validation for bucket name
	// 1. No consecutive periods or hyphens
	if strings.Contains(bucketName, "..") || strings.Contains(bucketName, "--") {
		return errors.New("Bucket name contains invalid characters")
	}

	// 2. No period-hyphen or hyphen-period
	if strings.Contains(bucketName, ".-") || strings.Contains(bucketName, "-.") {
		return errors.New("Bucket name has unexpected format or contains invalid characters")
	}

	// 3. No IP address format (e.g., 192.168.0.1)
	if ipAddress.MatchString(bucketName) {
		return errors.New("Bucket name cannot be an ip address")
	}

	return nil
}

// CheckValidBucketNameStrict - checks if we have a valid input bucket name.
// This is a stricter version.
// - http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingBucket.html
func CheckValidBucketNameStrict(bucketName string) (err error) {
	return checkBucketNameCommon(bucketName, true)
}

// CheckValidObjectNamePrefix - checks if we have a valid input object name prefix.
//   - http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingMetadata.html
func CheckValidObjectNamePrefix(objectName string) error {
	if len(objectName) > 1024 {
		return errors.New("Object name cannot be longer than 1024 characters")
	}
	if !utf8.ValidString(objectName) {
		return errors.New("Object name with non UTF-8 strings are not supported")
	}
	return nil
}

// CheckValidObjectName - checks if we have a valid input object name.
//   - http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingMetadata.html
func CheckValidObjectName(objectName string) error {
	if strings.TrimSpace(objectName) == "" {
		return errors.New("Object name cannot be empty")
	}
	return CheckValidObjectNamePrefix(objectName)
}
