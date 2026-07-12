package qcom

import (
	"context"
	"net"
	"time"

	"github.com/damonto/wwan-go/qcom/tlv"
)

// ServiceType represents QMI service types.
type ServiceType uint8

const (
	ServiceControl ServiceType = 0x00 // Control service
	ServiceWDS     ServiceType = 0x01 // Wireless Data Service
	ServiceDMS     ServiceType = 0x02 // Device Management Service
	ServiceNAS     ServiceType = 0x03 // Network Access Service
	ServiceCAT2    ServiceType = 0x0A // Card Application Toolkit service v2
	ServiceUIM     ServiceType = 0x0B // UIM service
	ServiceIMSS    ServiceType = 0x12 // IMS Settings service
	ServiceWDA     ServiceType = 0x1A // Wireless Data Administrative service
	ServiceIMSA    ServiceType = 0x21 // IMS Application service
	ServiceCAT     ServiceType = 0xE0 // Card Application Toolkit service v1
)

// MessageType represents QMI message types.
type MessageType uint8

const (
	MessageTypeRequest    MessageType = 0x00
	MessageTypeResponse   MessageType = 0x02
	MessageTypeIndication MessageType = 0x04
)

// MessageID represents QMI command message IDs.
type MessageID uint16

const (
	// CTL service commands
	MessageGetVersionInfo    MessageID = 0x0021
	MessageAllocateClientID  MessageID = 0x0022
	MessageReleaseClientID   MessageID = 0x0023
	MessageInternalProxyOpen MessageID = 0xFF00

	// WDS service commands
	MessageWDSStartNetworkInterface MessageID = 0x0020
	MessageWDSStopNetworkInterface  MessageID = 0x0021
	MessageWDSCreateProfile         MessageID = 0x0027
	MessageWDSDeleteProfile         MessageID = 0x0029
	MessageWDSGetProfileList        MessageID = 0x002A
	MessageWDSGetProfileSettings    MessageID = 0x002B
	MessageWDSGetRuntimeSettings    MessageID = 0x002D
	MessageWDSSetClientIPFamily     MessageID = 0x004D
	MessageWDSLegacyBindMuxDataPort MessageID = 0x0089
	MessageWDSBindMuxDataPort       MessageID = 0x00A2

	// WDA service commands
	MessageWDASetDataFormat MessageID = 0x0020
	MessageWDAGetDataFormat MessageID = 0x0021

	// DMS service commands
	MessageDMSSetEventReport   MessageID = 0x0001
	MessageDMSGetMSISDN        MessageID = 0x0024
	MessageDMSGetOperatingMode MessageID = 0x002D
	MessageDMSSetOperatingMode MessageID = 0x002E

	// NAS service commands
	MessageNASGetServingSystem MessageID = 0x0024
	MessageNASGetSysInfo       MessageID = 0x004D

	// IMSA service commands
	MessageIMSAGetRegistrationStatus MessageID = 0x0020
	MessageIMSAGetServiceStatus      MessageID = 0x0021

	// IMSS service commands
	MessageIMSSSetRegistrationManagerConfig MessageID = 0x0021
	MessageIMSSGetRegistrationManagerConfig MessageID = 0x0026

	// UIM service commands
	MessageReset                     MessageID = 0x0000
	MessageReadTransparent           MessageID = 0x0020
	MessageReadRecord                MessageID = 0x0021
	MessageWriteRecord               MessageID = 0x0023
	MessageGetFileAttributes         MessageID = 0x0024
	MessageRefreshRegister           MessageID = 0x002A
	MessageRefreshComplete           MessageID = 0x002C
	MessageRegisterEvents            MessageID = 0x002E
	MessagePowerOffSIM               MessageID = 0x0030
	MessagePowerOnSIM                MessageID = 0x0031
	MessageRefresh                   MessageID = 0x0033
	MessageChangeProvisioningSession MessageID = 0x0038
	MessageSendAPDU                  MessageID = 0x003B
	MessageOpenLogicalChannel        MessageID = 0x0042
	MessageCloseLogicalChannel       MessageID = 0x003F
	MessageGetATR                    MessageID = 0x0041
	MessageRefreshRegisterAll        MessageID = 0x0044
	MessageSwitchSlot                MessageID = 0x0046
	MessageGetSlotStatus             MessageID = 0x0047
	MessageSlotStatus                MessageID = 0x0048
	MessageGetCardStatus             MessageID = 0x002F
	MessageAuthenticate              MessageID = 0x0034

	// CAT/CAT2 service commands
	MessageCATSetEventReport       MessageID = 0x0001
	MessageCATEventReport          MessageID = 0x0001
	MessageCATGetServiceState      MessageID = 0x0020
	MessageCATSendTerminalResponse MessageID = 0x0021
	MessageSendEnvelope            MessageID = 0x0022
	MessageCATSendEnvelope         MessageID = 0x0022
	MessageCATEventConfirmation    MessageID = 0x0026
	MessageCATGetTerminalProfile   MessageID = 0x002C
	MessageCATSetConfiguration     MessageID = 0x002D
	MessageCATGetConfiguration     MessageID = 0x002E
)

// QMIResult represents the result code in QMI responses.
type QMIResult uint16

const (
	QMIResultSuccess QMIResult = 0x0000 // Success
	QMIResultFailure QMIResult = 0x0001 // Failure
)

// WDSIPFamily identifies the address family negotiated for an active WDS call.
type WDSIPFamily uint8

const (
	WDSIPFamilyIPv4 WDSIPFamily = 4
	WDSIPFamilyIPv6 WDSIPFamily = 6
)

// WDSIPPreference selects the address family requested when starting a WDS
// call. The zero value omits the optional QMI TLV and lets the modem use its
// default. QMI value 8 means unspecified; it is not an active dual-stack
// family value.
type WDSIPPreference uint8

const (
	WDSIPPreferenceDefault     WDSIPPreference = 0
	WDSIPPreferenceIPv4        WDSIPPreference = 4
	WDSIPPreferenceIPv6        WDSIPPreference = 6
	WDSIPPreferenceUnspecified WDSIPPreference = 8
)

// WDSCallType identifies the origin of a WDS packet-data call.
type WDSCallType uint8

const (
	WDSCallTypeLaptop WDSCallType = iota
	WDSCallTypeEmbedded
)

// WDSTechnologyPreference is the WDS technology preference bit mask.
type WDSTechnologyPreference uint8

const (
	WDSTechnologyPreference3GPP WDSTechnologyPreference = 1
)

// WDSSIOPort identifies a legacy modem SIO data port.
type WDSSIOPort uint16

const (
	WDSSIOPortA2MuxRMNET0 WDSSIOPort = 0x0E04 + iota
	WDSSIOPortA2MuxRMNET1
	WDSSIOPortA2MuxRMNET2
	WDSSIOPortA2MuxRMNET3
	WDSSIOPortA2MuxRMNET4
	WDSSIOPortA2MuxRMNET5
	WDSSIOPortA2MuxRMNET6
	WDSSIOPortA2MuxRMNET7
)

// DataEndpointType identifies the physical data transport endpoint.
type DataEndpointType uint32

const (
	DataEndpointReserved DataEndpointType = iota
	DataEndpointHSIC
	DataEndpointHSUSB
	DataEndpointPCIe
	DataEndpointEmbedded
	DataEndpointBAMDMUX
)

// DataEndpoint identifies a physical data channel exposed by the modem.
type DataEndpoint struct {
	Type        DataEndpointType
	InterfaceID uint32
}

// WDSDataEndpointType is kept for source compatibility.
// Deprecated: use DataEndpointType.
type WDSDataEndpointType = DataEndpointType

const (
	WDSDataEndpointReserved = DataEndpointReserved
	WDSDataEndpointHSIC     = DataEndpointHSIC
	WDSDataEndpointHSUSB    = DataEndpointHSUSB
	WDSDataEndpointPCIe     = DataEndpointPCIe
	WDSDataEndpointEmbedded = DataEndpointEmbedded
	WDSDataEndpointBAMDMUX  = DataEndpointBAMDMUX
)

// WDSDataEndpoint is kept for source compatibility.
// Deprecated: use DataEndpoint.
type WDSDataEndpoint = DataEndpoint

// WDALinkLayerProtocol identifies the frames exchanged on the modem data port.
type WDALinkLayerProtocol uint32

const (
	WDALinkLayerEthernet WDALinkLayerProtocol = 0x01
	WDALinkLayerRawIP    WDALinkLayerProtocol = 0x02
)

// WDAAggregationProtocol identifies a modem data aggregation format.
type WDAAggregationProtocol uint32

const (
	WDAAggregationDisabled WDAAggregationProtocol = iota
	WDAAggregationTLP
	WDAAggregationQCNCM
	WDAAggregationMBIM
	WDAAggregationRNDIS
	WDAAggregationQMAP
	WDAAggregationQMAPv2
	WDAAggregationQMAPv3
)

// WDAQoSHeaderFormat identifies the optional uplink QoS header layout.
type WDAQoSHeaderFormat uint32

const (
	WDAQoSHeaderReserved WDAQoSHeaderFormat = iota
	WDAQoSHeader6Bytes
	WDAQoSHeader8Bytes
)

// WDADataFormatConfig selects fields for WDA Set Data Format.
// Nil fields are omitted because every WDA data-format TLV is optional.
type WDADataFormatConfig struct {
	QoSEnabled                   *bool
	LinkLayerProtocol            *WDALinkLayerProtocol
	UplinkAggregation            *WDAAggregationProtocol
	DownlinkAggregation          *WDAAggregationProtocol
	NDPSignature                 *uint32
	DownlinkMaxDatagrams         *uint32
	DownlinkMaxSize              *uint32
	Endpoint                     *DataEndpoint
	QoSHeaderFormat              *WDAQoSHeaderFormat
	DownlinkMinimumPadding       *uint32
	TerminalEquipmentFlowControl *bool
}

// WDADataFormat contains data-format fields returned by the modem.
// A Known flag distinguishes an absent optional TLV from a zero value.
type WDADataFormat struct {
	QoSEnabled                        bool
	QoSEnabledKnown                   bool
	LinkLayerProtocol                 WDALinkLayerProtocol
	LinkLayerProtocolKnown            bool
	UplinkAggregation                 WDAAggregationProtocol
	UplinkAggregationKnown            bool
	DownlinkAggregation               WDAAggregationProtocol
	DownlinkAggregationKnown          bool
	NDPSignature                      uint32
	NDPSignatureKnown                 bool
	DownlinkMaxDatagrams              uint32
	DownlinkMaxDatagramsKnown         bool
	DownlinkMaxSize                   uint32
	DownlinkMaxSizeKnown              bool
	UplinkMaxDatagrams                uint32
	UplinkMaxDatagramsKnown           bool
	UplinkMaxSize                     uint32
	UplinkMaxSizeKnown                bool
	QoSHeaderFormat                   WDAQoSHeaderFormat
	QoSHeaderFormatKnown              bool
	DownlinkMinimumPadding            uint32
	DownlinkMinimumPaddingKnown       bool
	TerminalEquipmentFlowControl      bool
	TerminalEquipmentFlowControlKnown bool
}

// WDSMuxDataPort describes the logical data channel assigned to a WDS client.
type WDSMuxDataPort struct {
	Endpoint   *DataEndpoint
	MuxID      uint8
	Reversed   bool
	ClientType WDSClientType
}

type WDSClientType uint32

const (
	WDSClientTypeReserved WDSClientType = iota
	WDSClientTypeTethered
)

// WDSProfileType identifies a modem data-profile technology family.
type WDSProfileType uint8

const (
	WDSProfileType3GPP WDSProfileType = iota
	WDSProfileType3GPP2
	WDSProfileTypeEPC
)

// WDSPDPType identifies the packet data protocol stored in a 3GPP profile.
type WDSPDPType uint8

const (
	WDSPDPTypeIPv4 WDSPDPType = iota
	WDSPDPTypePPP
	WDSPDPTypeIPv6
	WDSPDPTypeIPv4v6
)

// WDSProfileID identifies a stored modem data profile.
type WDSProfileID struct {
	Type  WDSProfileType
	Index uint8
}

// WDSProfile is one entry returned by WDS Get Profile List.
type WDSProfile struct {
	ID   WDSProfileID
	Name string
}

// WDSProfileSettings contains selected optional WDS profile fields.
type WDSProfileSettings struct {
	ID WDSProfileID

	Name      string
	NameKnown bool
	APN       string
	APNKnown  bool

	PCSCFUsingPCO       bool
	PCSCFUsingPCOKnown  bool
	PCSCFUsingDHCP      bool
	PCSCFUsingDHCPKnown bool
	IMCN                bool
	IMCNKnown           bool
}

// WDSCallEndReason is the basic WDS call end reason returned by start-network.
type WDSCallEndReason uint16

const (
	WDSCallEndReasonGenericUnspecified WDSCallEndReason = 1
)

// WDSVerboseCallEndReasonType selects the namespace for a verbose WDS call end reason.
type WDSVerboseCallEndReasonType uint16

const (
	WDSVerboseCallEndReasonTypeMIP      WDSVerboseCallEndReasonType = 1
	WDSVerboseCallEndReasonTypeInternal WDSVerboseCallEndReasonType = 2
	WDSVerboseCallEndReasonTypeCM       WDSVerboseCallEndReasonType = 3
	WDSVerboseCallEndReasonType3GPP     WDSVerboseCallEndReasonType = 6
	WDSVerboseCallEndReasonTypePPP      WDSVerboseCallEndReasonType = 7
	WDSVerboseCallEndReasonTypeEHRPD    WDSVerboseCallEndReasonType = 8
	WDSVerboseCallEndReasonTypeIPv6     WDSVerboseCallEndReasonType = 9
)

// WDSVerboseCallEndReason is the structured WDS call failure reason.
type WDSVerboseCallEndReason struct {
	Type   WDSVerboseCallEndReasonType
	Reason int16
}

// WDSRuntimeSettingsMask selects fields returned by WDS Get Runtime Settings.
type WDSRuntimeSettingsMask uint32

const (
	WDSRuntimeMaskIPAddress     WDSRuntimeSettingsMask = 0x00000100
	WDSRuntimeMaskDNSAddress    WDSRuntimeSettingsMask = 0x00000010
	WDSRuntimeMaskGatewayInfo   WDSRuntimeSettingsMask = 0x00000200
	WDSRuntimeMaskMTU           WDSRuntimeSettingsMask = 0x00002000
	WDSRuntimeMaskPCSCFUsingPCO WDSRuntimeSettingsMask = 0x00000400
	WDSRuntimeMaskPCSCFServer   WDSRuntimeSettingsMask = 0x00000800
	WDSRuntimeMaskIPFamily      WDSRuntimeSettingsMask = 0x00008000
	WDSRuntimeMaskIMCNFlag      WDSRuntimeSettingsMask = 0x00010000

	WDSRuntimeRequestedIMSSettings = WDSRuntimeMaskIPAddress |
		WDSRuntimeMaskPCSCFUsingPCO |
		WDSRuntimeMaskPCSCFServer |
		WDSRuntimeMaskIPFamily |
		WDSRuntimeMaskIMCNFlag

	WDSRuntimeRequestedNetworkSettings = WDSRuntimeMaskDNSAddress |
		WDSRuntimeMaskIPAddress |
		WDSRuntimeMaskGatewayInfo |
		WDSRuntimeMaskMTU |
		WDSRuntimeMaskIPFamily
)

// WDSRuntimeSettings holds IMS PDN addressing and P-CSCF data from WDS.
type WDSRuntimeSettings struct {
	LocalIPv4        net.IP
	LocalIPv6        net.IP
	IPv4Gateway      net.IP
	IPv4SubnetMask   net.IP
	IPv6Gateway      net.IP
	IPv6PrefixLength uint8
	DNS              []net.IP
	MTU              uint32
	PCSCFIPs         []net.IP
	IPFamily         WDSIPFamily
	IMCN             bool
}

// DMSOperatingMode is the QMI DMS modem operating mode.
type DMSOperatingMode uint8

const (
	DMSOperatingModeOnline             DMSOperatingMode = 0x00
	DMSOperatingModeLowPower           DMSOperatingMode = 0x01
	DMSOperatingModeFactoryTest        DMSOperatingMode = 0x02
	DMSOperatingModeOffline            DMSOperatingMode = 0x03
	DMSOperatingModeResetting          DMSOperatingMode = 0x04
	DMSOperatingModeShuttingDown       DMSOperatingMode = 0x05
	DMSOperatingModePersistentLowPower DMSOperatingMode = 0x06
	DMSOperatingModeModeOnlyLowPower   DMSOperatingMode = 0x07
	DMSOperatingModeNetworkTestGW      DMSOperatingMode = 0x08
)

// NASSysInfo is the NAS system information used by IMS access selection.
type NASSysInfo struct {
	VoPSKnown     bool
	VoPSSupported bool
}

// NASRegistrationState is the network registration state reported by NAS.
type NASRegistrationState uint8

const (
	NASRegistrationNotRegistered NASRegistrationState = iota
	NASRegistrationRegistered
	NASRegistrationSearching
	NASRegistrationDenied
	NASRegistrationUnknown
)

// NASAttachState is a circuit-switched or packet-switched attach state.
type NASAttachState uint8

const (
	NASAttachUnknown NASAttachState = iota
	NASAttachAttached
	NASAttachDetached
)

// NASSelectedNetwork identifies the selected network family.
type NASSelectedNetwork uint8

const (
	NASSelectedNetworkUnknown NASSelectedNetwork = iota
	NASSelectedNetwork3GPP2
	NASSelectedNetwork3GPP
)

// NASRadioInterface identifies a radio interface currently in use.
type NASRadioInterface uint8

const (
	NASRadioInterfaceNoService NASRadioInterface = 0
	NASRadioInterfaceCDMA1X    NASRadioInterface = 1
	NASRadioInterfaceCDMAEVDO  NASRadioInterface = 2
	NASRadioInterfaceAMPS      NASRadioInterface = 3
	NASRadioInterfaceGSM       NASRadioInterface = 4
	NASRadioInterfaceUMTS      NASRadioInterface = 5
	NASRadioInterfaceLTE       NASRadioInterface = 8
)

// NASServingSystem contains the fields from NAS Get Serving System.
type NASServingSystem struct {
	RegistrationState NASRegistrationState
	CSAttachState     NASAttachState
	PSAttachState     NASAttachState
	SelectedNetwork   NASSelectedNetwork
	RadioInterfaces   []NASRadioInterface
}

// IMSRegistrationStatus is the QMI IMSA registration state.
type IMSRegistrationStatus uint32

const (
	IMSRegistrationStatusNotRegistered IMSRegistrationStatus = 0
	IMSRegistrationStatusRegistering   IMSRegistrationStatus = 1
	IMSRegistrationStatusRegistered    IMSRegistrationStatus = 2
)

// IMSServiceStatus is the QMI IMSA per-service availability state.
type IMSServiceStatus uint32

const (
	IMSServiceStatusNoService      IMSServiceStatus = 0
	IMSServiceStatusLimitedService IMSServiceStatus = 1
	IMSServiceStatusFullService    IMSServiceStatus = 2
)

// IMSServiceRAT is the access technology used by an IMS service.
type IMSServiceRAT uint32

const (
	IMSServiceRATWLAN  IMSServiceRAT = 0
	IMSServiceRATWWAN  IMSServiceRAT = 1
	IMSServiceRATIWLAN IMSServiceRAT = 2
)

// IMSAStatus contains IMS registration and VoIP service information from QMI IMSA.
type IMSAStatus struct {
	RegistrationKnown bool
	Registration      IMSRegistrationStatus
	FailureCodeKnown  bool
	FailureCode       uint16
	VoIPServiceKnown  bool
	VoIPService       IMSServiceStatus
	VoIPRATKnown      bool
	VoIPRAT           IMSServiceRAT
}

// IMSRegistered reports whether the modem is registered on IMS.
func (s IMSAStatus) IMSRegistered() bool {
	return s.RegistrationKnown && s.Registration == IMSRegistrationStatusRegistered
}

// VoLTERegistered reports whether IMS VoIP service is registered over WWAN.
func (s IMSAStatus) VoLTERegistered() bool {
	return s.IMSRegistered() &&
		s.VoIPServiceKnown && s.VoIPService == IMSServiceStatusFullService &&
		s.VoIPRATKnown && s.VoIPRAT == IMSServiceRATWWAN
}

type Request struct {
	Service       ServiceType
	ClientID      uint8
	TransactionID uint16
	MessageID     MessageID
	Timeout       time.Duration
	TLVs          tlv.TLVs
}

type Response struct {
	Service       ServiceType
	ClientID      uint8
	TransactionID uint16
	MessageID     MessageID
	TLVs          tlv.TLVs
}

// Indication is an unsolicited QMI message delivered outside a request/response
// transaction.
type Indication struct {
	Service       ServiceType
	ClientID      uint8
	TransactionID uint16
	MessageID     MessageID
	TLVs          tlv.TLVs
}

type Transport interface {
	Do(ctx context.Context, req Request) (Response, error)
	Close() error
}

// IndicationTransport extends Transport with best-effort indication delivery.
//
// Indications returns a channel for unsolicited messages matching service,
// clientID, and id. The channel is closed when ctx is done or the transport
// closes. Delivery is lossy: a slow subscriber may miss indications.
type IndicationTransport interface {
	Transport
	Indications(ctx context.Context, service ServiceType, clientID uint8, id MessageID) (<-chan Indication, error)
}

type FileStructure byte

const (
	FileStructureTransparent FileStructure = 0x41
	FileStructureLinearFixed FileStructure = 0x42
)

type FileType byte

const (
	FileTypeWorkingEF FileType = 0x21
	FileTypeDFOrADF   FileType = 0x38
)

type CardState byte

const (
	CardStateAbsent CardState = iota
	CardStatePresent
	CardStateError
)

type PhysicalCardState uint32

const (
	PhysicalCardStateUnknown PhysicalCardState = iota
	PhysicalCardStateAbsent
	PhysicalCardStatePresent
)

type SlotState uint32

const (
	SlotStateInactive SlotState = iota
	SlotStateActive
)

type CardProtocol uint32

const (
	CardProtocolUnknown CardProtocol = iota
	CardProtocolICC
	CardProtocolUICC
)

type QMIFileType byte

const (
	QMIFileTypeTransparent QMIFileType = iota
	QMIFileTypeCyclic
	QMIFileTypeLinearFixed
	QMIFileTypeDedicated
	QMIFileTypeMaster
)

type PINState byte

const (
	PINStateNotInitialized PINState = iota
	PINStateEnabledNotVerified
	PINStateEnabledVerified
	PINStateDisabled
	PINStateBlocked
	PINStatePermanentlyBlocked
)

type CardError byte

const (
	CardErrorUnknown CardError = iota
	CardErrorPowerDown
	CardErrorPoll
	CardErrorNoATRReceived
	CardErrorVoltageMismatch
	CardErrorParity
	CardErrorPossiblyRemoved
	CardErrorTechnical
)

type ApplicationType byte

const (
	ApplicationTypeUnknown ApplicationType = iota
	ApplicationTypeSIM
	ApplicationTypeUSIM
	ApplicationTypeRUIM
	ApplicationTypeCSIM
	ApplicationTypeISIM
)

type ApplicationState byte

const (
	ApplicationStateUnknown ApplicationState = iota
	ApplicationStateDetected
	ApplicationStatePIN1OrUPINRequired
	ApplicationStatePUK1OrUPINRequired
	ApplicationStateCheckPersonalization
	ApplicationStatePIN1Blocked
	ApplicationStateIllegal
	ApplicationStateReady
)

type PersonalizationState byte

const (
	PersonalizationStateUnknown PersonalizationState = iota
	PersonalizationStateInProgress
	PersonalizationStateReady
	PersonalizationStateCodeRequired
	PersonalizationStatePUKCodeRequired
	PersonalizationStatePermanentlyBlocked
)

type PersonalizationFeature byte

const (
	PersonalizationFeatureGWNetwork PersonalizationFeature = iota
	PersonalizationFeatureGWNetworkSubset
	PersonalizationFeatureGWServiceProvider
	PersonalizationFeatureGWCorporate
	PersonalizationFeatureGWUIM
	PersonalizationFeatureOneXNetworkType1
	PersonalizationFeatureOneXNetworkType2
	PersonalizationFeatureOneXHRPD
	PersonalizationFeatureOneXServiceProvider
	PersonalizationFeatureOneXCorporate
	PersonalizationFeatureOneXRUIM
	PersonalizationFeatureGWServiceProviderName
	PersonalizationFeatureGWSPAndEHPLMN
	PersonalizationFeatureGWICCID
	PersonalizationFeatureGWIMPI
	PersonalizationFeatureGWNetworkSubsetServiceProvider
	PersonalizationFeatureGWCarrier
)

type CATConfigMode uint8

const (
	CATConfigDisabled      CATConfigMode = 0x00
	CATConfigGobi          CATConfigMode = 0x01
	CATConfigAndroid       CATConfigMode = 0x02
	CATConfigDecoded       CATConfigMode = 0x03
	CATConfigDecodedPull   CATConfigMode = 0x04
	CATConfigCustomRaw     CATConfigMode = 0x05
	CATConfigCustomDecoded CATConfigMode = 0x06
)

type Session uint8

const (
	SessionPrimaryGWProvisioning Session = 0

	SessionNonProvisioningSlot1 Session = 4
	SessionNonProvisioningSlot2 Session = 5
	SessionCardSlot1            Session = 6
	SessionCardSlot2            Session = 7

	SessionNonProvisioningSlot3 Session = 16
	SessionNonProvisioningSlot4 Session = 17
	SessionNonProvisioningSlot5 Session = 18
	SessionCardSlot3            Session = 19
	SessionCardSlot4            Session = 20
	SessionCardSlot5            Session = 21
)

type FileAttributes struct {
	FileStructure FileStructure
	FileType      FileType
	RecordSize    uint16
	RecordCount   uint16
	FileSize      uint16
}

type File struct {
	Session Session
	AID     []byte
	Path    []byte
}

type TransparentRead struct {
	File   File
	Offset uint16
	Length uint16
}

type RecordRead struct {
	File   File
	Record uint16
	Length uint16
}

type RecordWrite struct {
	File   File
	Record uint16
	Data   []byte
}

type AuthContext byte

const (
	AuthContext3G     AuthContext = 3
	AuthContextIMSAKA AuthContext = 11
)

type AuthenticateRequest struct {
	Session Session
	AID     []byte
	Context AuthContext
	Rand    []byte
	AUTN    []byte
}

type EnvelopeResponse struct {
	SW1  byte
	SW2  byte
	Data []byte
}

type PowerOnSIMRequest struct {
	Slot                uint8
	IgnoreHotSwapSwitch bool
}

type ChangeProvisioningSessionRequest struct {
	Session  Session
	Activate bool
	Slot     uint8
	AID      []byte
}

type OpenLogicalChannelRequest struct {
	AID []byte
}

type OpenLogicalChannelResponse struct {
	Channel uint8
}

type CloseLogicalChannelRequest struct {
	Channel uint8
}

type CloseLogicalChannelResponse struct{}

type SendAPDURequest struct {
	Command []byte
}

type SendAPDUResponse struct {
	Response []byte
}

type RefreshStage uint8

const (
	RefreshStageWaitForOK RefreshStage = iota
	RefreshStageStart
	RefreshStageEndWithSuccess
	RefreshStageEndWithFailure
)

type RefreshMode uint8

const (
	RefreshModeReset RefreshMode = iota
	RefreshModeInit
	RefreshModeInitFCN
	RefreshModeFCN
	RefreshModeInitFullFCN
	RefreshModeApplicationReset
	RefreshMode3GReset
)

type RefreshFile struct {
	FileID uint16
	Path   []byte
}

type RefreshEvent struct {
	Stage   RefreshStage
	Mode    RefreshMode
	Session Session
	AID     []byte
	Files   []RefreshFile
}

type RefreshRegisterRequest struct {
	Session     Session
	AID         []byte
	VoteForInit bool
	Files       []RefreshFile
}
