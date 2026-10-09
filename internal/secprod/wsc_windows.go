package secprod

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
)

// GUIDs and vtable layout from iwscapi.h of the Windows SDK 10.0.26100.0
var (
	clsidWSCProductList = ole.NewGUID("{17072F7B-9ABE-4A74-A261-1EB76B55107A}")
	iidIWSCProductList  = ole.NewGUID("{722A338C-6E8E-4E72-AC27-1417FB0C81C2}")
)

// WSC_SECURITY_PROVIDER from wscapi.h
var wscProviders = map[string]uint32{
	KindFirewall:    0x1,
	KindAntivirus:   0x4,
	KindAntispyware: 0x8,
}

// IWSCProductList and IWscProduct derive from IDispatch: slots 0-2 are IUnknown, 3-6 IDispatch,
// their own methods follow in header order
const (
	slotListInitialize = 7
	slotListCount      = 8
	slotListItem       = 9

	slotProductName            = 7
	slotProductState           = 8
	slotProductSignatureStatus = 9
	slotProductRemediationPath = 10
	slotProductStateTimestamp  = 11
)

// wscProducts reads Security Center through IWSCProductList, the documented API: go-ole has no
// binding for it, so its methods are called through the vtable as iwscapi.h declares them
func wscProducts(ctx context.Context, kind string) ([]Product, error) {
	provider, ok := wscProviders[kind]
	if !ok {
		return nil, fmt.Errorf("unknown product kind %q", kind)
	}
	return withCOM(ctx, func() ([]Product, error) {
		list, err := ole.CreateInstance(clsidWSCProductList, iidIWSCProductList)
		if err != nil {
			return nil, fmt.Errorf("CoCreateInstance(WSCProductList): %w", err)
		}
		defer list.Release()

		hr, _, _ := syscall.SyscallN(method(list, slotListInitialize), uintptr(unsafe.Pointer(list)), uintptr(provider))
		if err := hresult(hr); err != nil {
			return nil, fmt.Errorf("IWSCProductList::Initialize: %w", err)
		}
		var n int32
		if err := get(list, slotListCount, unsafe.Pointer(&n)); err != nil {
			return nil, fmt.Errorf("IWSCProductList::get_Count: %w", err)
		}
		products := make([]Product, 0, max(n, 0))
		for i := range n {
			var item *ole.IUnknown
			hr, _, _ = syscall.SyscallN(method(list, slotListItem), uintptr(unsafe.Pointer(list)), uintptr(i), uintptr(unsafe.Pointer(&item)))
			if err := hresult(hr); err != nil {
				return nil, fmt.Errorf("IWSCProductList::get_Item(%d): %w", i, err)
			}
			if item == nil {
				return nil, fmt.Errorf("IWSCProductList::get_Item(%d): no product", i)
			}
			p, err := wscProduct(item, kind)
			item.Release()
			if err != nil {
				return nil, err
			}
			products = append(products, p)
		}
		return withoutWindowsFirewall(products), nil
	})
}

func wscProduct(item *ole.IUnknown, kind string) (Product, error) {
	p := Product{Kind: kind, Signature: SignatureUnknown, Source: SourceWSC}
	var state int32
	if err := get(item, slotProductState, unsafe.Pointer(&state)); err != nil {
		return p, fmt.Errorf("IWscProduct::get_ProductState: %w", err)
	}
	p.State = wscState(state)
	name, err := getString(item, slotProductName)
	if err != nil {
		return p, fmt.Errorf("IWscProduct::get_ProductName: %w", err)
	}
	p.Name = name
	// the rest is detail: a product that fails to report it is still listed, signatures then unknown
	var signature int32
	if get(item, slotProductSignatureStatus, unsafe.Pointer(&signature)) == nil {
		p.Signature = wscSignature(signature)
	}
	p.Path, _ = getString(item, slotProductRemediationPath)
	p.Timestamp, _ = getString(item, slotProductStateTimestamp)
	return p, nil
}

func method(obj *ole.IUnknown, slot int) uintptr {
	return *(*uintptr)(unsafe.Add(unsafe.Pointer(obj.RawVTable), slot*int(unsafe.Sizeof(uintptr(0)))))
}

// get calls a property getter, HRESULT get_X(This, T *out): out stays an unsafe.Pointer up to the
// syscall so that the garbage collector keeps tracking it
func get(obj *ole.IUnknown, slot int, out unsafe.Pointer) error {
	hr, _, _ := syscall.SyscallN(method(obj, slot), uintptr(unsafe.Pointer(obj)), uintptr(out))
	return hresult(hr)
}

// getString reads a BSTR property, which the callee allocates and the caller frees
func getString(obj *ole.IUnknown, slot int) (string, error) {
	var s *uint16
	if err := get(obj, slot, unsafe.Pointer(&s)); err != nil {
		return "", err
	}
	defer func() { _ = ole.SysFreeString((*int16)(unsafe.Pointer(s))) }()
	return ole.BstrToString(s), nil
}

func hresult(hr uintptr) error {
	if int32(hr) < 0 {
		return ole.NewError(hr)
	}
	return nil
}
