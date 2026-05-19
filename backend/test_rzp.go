package main

import (
	"encoding/json"
	"fmt"
	razorpay "github.com/razorpay/razorpay-go"
)

func main() {
	client := razorpay.NewClient("rzp_live_Sg6eGce51LkW4G", "qLSsJGP8BXuUElOoB9WzZVum")
	body, err := client.Plan.Fetch("plan_SqxJeHlN9W4Toj", nil, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	b, _ := json.MarshalIndent(body, "", "  ")
	fmt.Println(string(b))
}
