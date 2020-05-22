/*
Copyright © 2020 Chen Yang <betterchen@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"errors"
	"fmt"

	"github.com/betterchen/client-go-examples/pkg/cronjob"
	"github.com/betterchen/client-go-examples/pkg/util"
	"github.com/spf13/cobra"
)

type actionEnum uint8

const (
	_apply = iota // create or update cronjob/s
	_get
	_event
	_list
	_delete
)

type cronjobOption struct {
	cmd    *cobra.Command
	args   []string
	client util.ClientInterface

	namespace string
}

func (o *cronjobOption) complete(cmd *cobra.Command, args []string) (err error) {
	ctxName, err := cmd.Flags().GetString("context")
	if err != nil {
		return err
	}
	if len(ctxName) == 0 {
		return errors.New("undefined context name")
	}

	o.namespace, err = cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}
	if len(o.namespace) == 0 {
		return fmt.Errorf("undefined namespace")
	}

	if o.client, err = util.NewClientSet(ctxName); err != nil {
		return err
	}

	o.cmd = cmd
	o.args = args

	return
}

func (o cronjobOption) run(action actionEnum) error {

	switch action {
	case _apply:
		fp, err := o.cmd.LocalFlags().GetString("file")
		if err != nil {
			return err
		}
		if len(fp) == 0 {
			return fmt.Errorf("invalid file path: %s", fp)
		}

		err = cronjob.CreateOrUpdateCronJobByYAML(o.client, fp)
		if err != nil {
			return err
		}
	case _get:
		if len(o.args) == 0 || len(o.args[0]) == 0 {
			return fmt.Errorf("undefined name")
		}
		cj, err := cronjob.GetCronJob(o.client, o.namespace, o.args[0])
		if err != nil {
			return err
		}
		fmt.Println(cj)
	case _event:
		if len(o.args) == 0 || len(o.args[0]) == 0 {
			return fmt.Errorf("undefined name")
		}
		events, err := cronjob.GetCronJobEvents(o.client, o.namespace, o.args[0])
		if err != nil {
			return err
		}
		fmt.Println(events)
	case _list:
		cjs, err := cronjob.ListCronJob(o.client, o.namespace)
		if err != nil {
			return err
		}
		fmt.Println(cjs)
	case _delete:
		if len(o.args) == 0 || len(o.args[0]) == 0 {
			return fmt.Errorf("undefined name")
		}
		if err := cronjob.DeleteCronJob(o.client, o.namespace, o.args[0]); err != nil {
			return err
		}
		fmt.Printf("deleted CronJob %s", o.args[0])
	default:
		return errors.New("invalid action")
	}

	return nil
}

// cronjobCmd represents the cronjob command
var cronjobCmd = &cobra.Command{
	Use:   "cronjob",
	Short: "CronJob ops via client-go",
	Long:  ``,
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "creates or updates a cronjob",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		opt := cronjobOption{}
		if err := opt.complete(cmd, args); err != nil {
			fmt.Println(err.Error())
			return
		}
		if err := opt.run(_apply); err != nil {
			fmt.Println(err.Error())
			return
		}
	},
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get the info about a cronjob",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		opt := cronjobOption{}
		if err := opt.complete(cmd, args); err != nil {
			fmt.Println(err.Error())
			return
		}
		if err := opt.run(_get); err != nil {
			fmt.Println(err.Error())
			return
		}
	},
}

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "list past events of a cronjob",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		opt := cronjobOption{}
		if err := opt.complete(cmd, args); err != nil {
			fmt.Println(err.Error())
			return
		}
		if err := opt.run(_event); err != nil {
			fmt.Println(err.Error())
			return
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list running cronjob",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		opt := cronjobOption{}
		if err := opt.complete(cmd, args); err != nil {
			fmt.Println(err.Error())
			return
		}
		if err := opt.run(_list); err != nil {
			fmt.Println(err.Error())
			return
		}
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a cronjob",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		opt := cronjobOption{}
		if err := opt.complete(cmd, args); err != nil {
			fmt.Println(err.Error())
			return
		}
		if err := opt.run(_delete); err != nil {
			fmt.Println(err.Error())
			return
		}
	},
}

func init() {
	getCmd.AddCommand(eventCmd)
	cronjobCmd.AddCommand(applyCmd, getCmd, listCmd, deleteCmd)
	rootCmd.AddCommand(cronjobCmd)

	applyCmd.Flags().StringP("file", "f", "", "input yaml file address")
	cronjobCmd.PersistentFlags().StringP("namespace", "n", "", "working namespace")
}
