package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"github.com/vmware/go-vcloud-director/v2/util"
)

// Initialize connection to vCloud Director
func initVCDClient(vcdUrl string, orgName string, apiToken string) (*govcd.VCDClient, error) {

	// Enable debugging to get more insight into requests
	util.EnableLogging = true

	// Create a new VCD client
	url, err := url.ParseRequestURI(vcdUrl + "/api")
	if err != nil {
		return nil, err
	}

	client := govcd.NewVCDClient(*url, false)
	_, err = client.SetApiToken(orgName, apiToken)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func deleteVAppByName(client *govcd.VCDClient, orgName, vdcName, vAppName string) error {
	// Get the organization by name
	org, err := client.GetOrgByName(orgName)
	if err != nil {
		return fmt.Errorf("error getting organization: %v", err)
	}

	// Get the VDC by name
	vdc, err := org.GetVDCByName(vdcName, false)
	if err != nil {
		return fmt.Errorf("error getting VDC: %v", err)
	}

	// Find the vApp by name
	vApp, err := vdc.GetVAppByName(vAppName, true)
	if err != nil {
		if strings.HasPrefix(err.Error(), "[ENF] entity not found") {
			return nil
		}
		return err
	}

	fmt.Printf("vApp state: '%s'\n", types.VAppStatuses[vApp.VApp.Status])

	if types.VAppStatuses[vApp.VApp.Status] != "RESOLVED" {
		return fmt.Errorf("vApp should be in RESOLVED state after HTTP500 error issue")
	}

	//Delete the vApp
	fmt.Printf("Deleting vApp '%s'...\n", vAppName)
	task, err := vApp.Delete()
	if err != nil {
		return fmt.Errorf("error deleting vApp: %v", err)
	}

	//Wait for the delete task to complete
	if err := task.WaitTaskCompletion(); err != nil {
		return fmt.Errorf("error waiting for delete task completion: %v", err)
	}

	fmt.Printf("vApp '%s' successfully deleted.\n", vAppName)
	return nil
}

func deleteCapvcdRdeByName(client *govcd.VCDClient, rdeName string) error {
	
	capvcdRdeType, err := client.GetRdeType("vmware", "capvcdCluster", "1.3.0")
	if err != nil {
		return fmt.Errorf("unable to get CAPVCD Cluster Rde Type v1.3.0: %v", err)
	}

	rdes, err := capvcdRdeType.GetRdesByName(rdeName)
	if err != nil {
		if strings.HasPrefix(err.Error(), "[ENF] entity not found") {
			return nil
		}
		return err
	}
	if len(rdes) > 1 {
		return fmt.Errorf("more then one RDE with name '%s' has been found (very strange situation)", rdeName)
	}

	if err := rdes[0].Delete(); err != nil {
		return fmt.Errorf("error deleting CAPVCD RDE with name %s: %v", rdeName, err)
	}
	return nil
}

// Main function
func main() {

	var (
		vcdUrl   string
		orgName  string
		vdcName  string
		apiToken string
		name string
	)

	flag.StringVar(&vcdUrl, "url", "", "VMWare vCD URL")
	flag.StringVar(&orgName, "vorg", "", "VMWare vCD organisation name")
	flag.StringVar(&vdcName, "vdc", "", "VMWare vCD virtual DC name")
	flag.StringVar(&apiToken, "token", os.Getenv("VCD_API_TOKEN"), "VMWare vCD API key")
	flag.StringVar(&name, "name", "", "Tanzu cluster name")
	flag.Parse()

	vcdClient, err := initVCDClient(vcdUrl, orgName, apiToken)
	if err != nil {
		log.Fatalf("Error initializing vCloud Director client: %v", err)
	}

	if err := deleteCapvcdRdeByName(vcdClient, name); err != nil {
		log.Fatalf("Error deleting CAPVCD RDE: %v", err)
	}

	if err := deleteVAppByName(vcdClient, orgName, vdcName, name); err != nil {
		log.Fatalf("Error deleting vApp: %v", err)
	}

}
