package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes"
	si "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	copyExample = `
	# copy a single secret from the origin ns to the destination ns
	%[1]s copy-secret origin-namespace destination-namespace secret-name

	# copy all secrets from the origin ns to the destination ns
	%[1]s copy-secret origin-namespace destination-namespace --all
`
)

type CopySecretOptions struct {
	configFlags *genericclioptions.ConfigFlags

	secrets       []string
	allSecrets    bool
	originNS      string
	destinationNS string
	oSecrets      []*apiv1.Secret
	verbose       bool
	client        *kubernetes.Clientset

	genericiooptions.IOStreams
}

func NewCopySecretOptions(streams genericiooptions.IOStreams, client *kubernetes.Clientset) *CopySecretOptions {
	return &CopySecretOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
		client:      client,
	}
}

func NewCmdCopySecret(streams genericiooptions.IOStreams, client *kubernetes.Clientset) *cobra.Command {
	o := NewCopySecretOptions(streams, client)
	cmd := &cobra.Command{
		Use:          "copy-secret [flags]",
		Short:        "Copy secret(s) from one namespace to another",
		Example:      fmt.Sprintf(copyExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			// based on the example at https://github.com/kubernetes/sample-cli-plugin/blob/master/pkg/cmd/ns.go
			if err := o.SourceSecrets(c); err != nil {
				return err
			}
			o.Run(c)
			return nil
		},
	}
	cmd.Flags().StringVar(&o.originNS, "origin", o.originNS, "the namespace name to copy secrets from")
	cmd.Flags().StringVar(&o.destinationNS, "destination", o.destinationNS, "the namespace name to copy secrets to")
	cmd.Flags().StringSliceVar(&o.secrets, "secret", o.secrets, "a comma separated list (can be one) of secrets to copy")
	cmd.Flags().BoolVar(&o.allSecrets, "all", o.allSecrets, "if true, copy all secrets from the origin to the destination")
	cmd.Flags().BoolVar(&o.verbose, "verbose", o.verbose, "additional output for debugging")
	_ = cmd.MarkFlagRequired("origin")
	_ = cmd.MarkFlagRequired("destination")
	cmd.MarkFlagsOneRequired("secret", "all")
	cmd.MarkFlagsMutuallyExclusive("secret", "all")
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

// SourceSecrets prepares all obtainable secrets from the origin ns
func (o *CopySecretOptions) SourceSecrets(cmd *cobra.Command) error {
	var err error
	o.allSecrets, err = cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}

	if !o.allSecrets {
		if o.verbose {
			_, _ = fmt.Fprintf(o.Out, "getting secrets %s\n", o.secrets)
		}
		var errList []string
		for _, s := range o.secrets {
			err = o.AddSingleSecret(cmd.Context(), o.originNS, s)
			if err != nil {
				errList = append(errList, s)
			}
		}
		if len(errList) > 0 {
			_, _ = fmt.Fprintf(o.Out, "%q could not be found in the origin ns so they will be skipped\n", errList)
		}
	} else {
		if o.verbose {
			_, _ = fmt.Fprintf(o.Out, "getting all secrets in the %q ns\n", o.originNS)
		}
		err = o.AllSecretsFrom(cmd.Context(), o.originNS)
		if err != nil {
			_, _ = fmt.Fprintf(o.Out, "error getting all secrets from %q: %s", o.originNS, err)
			// assume we don't have any secrets to copy and return the error
			return err
		}
	}
	if o.verbose {
		for _, s := range o.oSecrets {
			_, _ = fmt.Fprintf(o.Out, "%q secret found\n", s.Name)
		}
	}
	return nil
}

func (o *CopySecretOptions) Run(cmd *cobra.Command) {
	for _, secret := range o.oSecrets {
		s := &apiv1.Secret{
			TypeMeta: secret.TypeMeta,
			ObjectMeta: v1.ObjectMeta{
				Name:      secret.ObjectMeta.Name,
				Namespace: o.destinationNS,
			},
			Immutable:  secret.Immutable,
			Data:       secret.Data,
			StringData: secret.StringData,
			Type:       secret.Type,
		}
		err := o.PutSecret(cmd.Context(), o.destinationNS, s)
		if err != nil {
			_, _ = fmt.Fprintf(o.Out, "error putting secret %q in %q ns: failed with error %s\n", s.Name, o.destinationNS, err)
		}
	}
}

// AllSecretsFrom gets all secret objects from the provided namespace and loads them to the list of secrets
func (o *CopySecretOptions) AllSecretsFrom(ctx context.Context, ns string) error {
	secretInterface := o.client.CoreV1().Secrets(ns)
	var err error
	o.oSecrets, err = allSecretsFrom(ctx, secretInterface)
	if err != nil {
		if o.verbose {
			_, _ = fmt.Fprintf(o.Out, "error listing all secrets: %s\n", err)
		}
		return err
	}
	return nil
}

func allSecretsFrom(ctx context.Context, i si.SecretInterface) ([]*apiv1.Secret, error) {
	var secrets []*apiv1.Secret
	secretList, err := i.List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, secret := range secretList.Items {
		s := &apiv1.Secret{
			TypeMeta:   secret.TypeMeta,
			ObjectMeta: secret.ObjectMeta,
			Immutable:  secret.Immutable,
			Data:       secret.Data,
			StringData: secret.StringData,
			Type:       secret.Type,
		}
		secrets = append(secrets, s)
	}
	return secrets, nil
}

// AddSingleSecret adds the named secret from the provided namespace to the list of secrets
func (o *CopySecretOptions) AddSingleSecret(ctx context.Context, ns string, name string) error {
	secretInterface := o.client.CoreV1().Secrets(ns)
	s, err := getSingleSecret(ctx, secretInterface, name)
	if err != nil {
		if o.verbose {
			_, _ = fmt.Fprintf(o.Out, "error getting secret %q from ns %q: %s", name, ns, err)
		}
		return err
	}
	o.oSecrets = append(o.oSecrets, s)
	return nil
}

func getSingleSecret(ctx context.Context, i si.SecretInterface, name string) (*apiv1.Secret, error) {
	s, err := i.Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return s, nil
}

// PutSecret will create the provided secret in the destination ns
func (o *CopySecretOptions) PutSecret(ctx context.Context, dest string, secret *apiv1.Secret) error {
	if o.verbose {
		_, _ = fmt.Fprintf(o.Out, "creating secret %q in %q ns\n", secret.Name, dest)
	}
	secretInterface := o.client.CoreV1().Secrets(dest)
	err := putSecret(ctx, secretInterface, secret)
	if err != nil {
		if o.verbose {
			_, _ = fmt.Fprintf(o.Out, "error creating secret %q in %q ns", secret.Name, dest)
		}
		return err
	}
	return nil
}

func putSecret(ctx context.Context, i si.SecretInterface, secret *apiv1.Secret) error {
	_, err := i.Create(ctx, secret, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
