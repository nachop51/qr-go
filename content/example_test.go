package content_test

import (
	"fmt"

	"github.com/nachop51/qr-go/content"
)

func ExampleWiFi() {
	fmt.Println(content.WiFi{SSID: "CoffeeShop", Pass: "latte123"}.String())
	// Output: WIFI:S:CoffeeShop;T:WPA;P:latte123;;
}

func ExampleTel() {
	fmt.Println(content.Tel("+15551234567"))
	// Output: tel:+15551234567
}

func ExampleGeo() {
	fmt.Println(content.Geo(48.8584, 2.2945))
	// Output: geo:48.8584,2.2945
}

func ExampleEmail() {
	fmt.Println(content.Email("hi@example.com", "Hello", "Body text"))
	// Output: mailto:hi@example.com?body=Body%20text&subject=Hello
}
