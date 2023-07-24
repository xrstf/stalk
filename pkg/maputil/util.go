// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package maputil

import "k8s.io/apimachinery/pkg/util/json"

func toJSON(obj interface{}) string {
	encoded, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
