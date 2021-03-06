package google

import (
	"fmt"
	"io/ioutil"

	"github.com/appscode/go/types"
	"github.com/spf13/pflag"
	"k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Options struct {
	ServiceAccountJsonFile string
	AdminEmail             string
}

func NewOptions() Options {
	return Options{}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ServiceAccountJsonFile, "google.sa-json-file", o.ServiceAccountJsonFile, "Path to Google service account json file")
	fs.StringVar(&o.AdminEmail, "google.admin-email", o.AdminEmail, "Email of G Suite administrator")
}

func (o *Options) Validate() []error {
	return nil
}

func (o Options) IsSet() bool {
	return o.ServiceAccountJsonFile != ""
}

func (o Options) Apply(d *v1beta1.Deployment) (extraObjs []runtime.Object, err error) {
	if !o.IsSet() {
		return nil, nil // nothing to apply
	}

	container := d.Spec.Template.Spec.Containers[0]

	// create auth secret
	sa, err := ioutil.ReadFile(o.ServiceAccountJsonFile)
	if err != nil {
		return nil, err
	}
	authSecret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "guard-google-auth",
			Namespace: d.Namespace,
			Labels:    d.Labels,
		},
		Data: map[string][]byte{
			"sa.json": sa,
		},
	}
	extraObjs = append(extraObjs, authSecret)

	// mount auth secret into deployment
	volMount := core.VolumeMount{
		Name:      authSecret.Name,
		MountPath: "/etc/guard/auth/google",
	}
	container.VolumeMounts = append(container.VolumeMounts, volMount)

	vol := core.Volume{
		Name: authSecret.Name,
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				SecretName:  authSecret.Name,
				DefaultMode: types.Int32P(0555),
			},
		},
	}
	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, vol)

	// use auth secret in container[0] args
	args := container.Args
	if o.ServiceAccountJsonFile != "" {
		args = append(args, "--google.sa-json-file=/etc/guard/auth/google/sa.json")
	}
	if o.AdminEmail != "" {
		args = append(args, fmt.Sprintf("--google.admin-email=%s", o.AdminEmail))
	}

	return extraObjs, nil
}
