// Pulled from https://github.com/aws/aws-sdk-go-v2/issues/1222
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	// "github.com/pkg/browser"

	"github.com/chromedp/chromedp"
)

var (
	startURL  string
	accountID string
	roleName  string
)

func main() {
	flag.StringVar(&startURL, "start-url", "", "AWS SSO Start URL")
	flag.StringVar(&accountID, "account-id", "", "AWS Account ID to fetch credentials for")
	flag.StringVar(&roleName, "role-name", "", "Role Name to assume while fetching credentials")
	flag.Parse()
	if startURL == "" || accountID == "" || roleName == "" {
		flag.Usage()
		os.Exit(1)
	}
	// load default aws config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Println(err)
	}
	// create sso oidc client to trigger login flow
	ssooidcClient := ssooidc.NewFromConfig(cfg)
	if err != nil {
		fmt.Println(err)
	}
	// register your client which is triggering the login flow
	register, err := ssooidcClient.RegisterClient(context.TODO(), &ssooidc.RegisterClientInput{
		ClientName: aws.String("sample-client-name"),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso-portal:*"},
	})
	if err != nil {
		fmt.Println(err)
	}
	// authorize your device using the client registration response
	deviceAuth, err := ssooidcClient.StartDeviceAuthorization(context.TODO(), &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		StartUrl:     aws.String(startURL),
	})
	if err != nil {
		fmt.Println(err)
	}
	// trigger OIDC login. open browser to login. close tab once login is done. press enter to continue
	url := aws.ToString(deviceAuth.VerificationUriComplete)
	fmt.Printf("If browser is not opened automatically, please open link:\n%v\n", url)
	// err = browser.OpenURL(url)
	// Foreground for debugging
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))...)
	defer cancel()
	browser, cancel := chromedp.NewContext(ctx)
	defer cancel()
	err = chromedp.Run(browser,
		chromedp.Navigate(*aws.String(startURL)),
		chromedp.WaitVisible(`cli_login_button`),
		chromedp.Click(`cli_login_button`, chromedp.NodeVisible),
	)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Press ENTER key once login is done")
	_ = bufio.NewScanner(os.Stdin).Scan()
	// generate sso token
	token, err := ssooidcClient.CreateToken(context.TODO(), &ssooidc.CreateTokenInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		DeviceCode:   deviceAuth.DeviceCode,
		GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
	})
	if err != nil {
		fmt.Println(err)
	}
	// create sso client
	ssoClient := sso.NewFromConfig(cfg)
	// list accounts [ONLY provided for better example coverage]
	fmt.Println("Fetching list of all accounts for user")
	accountPaginator := sso.NewListAccountsPaginator(ssoClient, &sso.ListAccountsInput{
		AccessToken: token.AccessToken,
	})
	for accountPaginator.HasMorePages() {
		x, err := accountPaginator.NextPage(context.TODO())
		if err != nil {
			fmt.Println(err)
		}
		for _, y := range x.AccountList {
			fmt.Println("-------------------------------------------------------")
			fmt.Printf("\nAccount ID: %v Name: %v Email: %v\n", aws.ToString(y.AccountId), aws.ToString(y.AccountName), aws.ToString(y.EmailAddress))

			// list roles for a given account [ONLY provided for better example coverage]
			fmt.Printf("\n\nFetching roles of account %v for user\n", aws.ToString(y.AccountId))
			rolePaginator := sso.NewListAccountRolesPaginator(ssoClient, &sso.ListAccountRolesInput{
				AccessToken: token.AccessToken,
				AccountId:   y.AccountId,
			})
			for rolePaginator.HasMorePages() {
				z, err := rolePaginator.NextPage(context.TODO())
				if err != nil {
					fmt.Println(err)
				}
				for _, p := range z.RoleList {
					fmt.Printf("Account ID: %v Role Name: %v\n", aws.ToString(p.AccountId), aws.ToString(p.RoleName))
				}
			}

		}
	}
	fmt.Println("-------------------------------------------------------")
	// exchange token received during oidc flow to fetch actual aws access keys
	fmt.Printf("\n\nFetching credentails for role %v of account %v for user\n", roleName, accountID)
	credentials, err := ssoClient.GetRoleCredentials(context.TODO(), &sso.GetRoleCredentialsInput{
		AccessToken: token.AccessToken,
		AccountId:   aws.String(accountID),
		RoleName:    aws.String(roleName),
	})
	if err != nil {
		fmt.Println(err)
	}
	// printing access key to show how they are accessed
	fmt.Printf("\n\nPriting aws access keysz")
	fmt.Println("Access key id: ", aws.ToString(credentials.RoleCredentials.AccessKeyId))
	fmt.Println("Secret access key: ", aws.ToString(credentials.RoleCredentials.SecretAccessKey))
	fmt.Println("Expiration: ", aws.ToInt64(&credentials.RoleCredentials.Expiration))
	fmt.Println("Session token: ", aws.ToString(credentials.RoleCredentials.SessionToken))
}
