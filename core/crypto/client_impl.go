/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package crypto

import (
	"errors"

	"github.com/hyperledger/fabric/core/crypto/primitives"
	"github.com/hyperledger/fabric/core/crypto/utils"
	obc "github.com/hyperledger/fabric/protos"
)

type clientImpl struct {
	*nodeImpl

	isInitialized bool

	// Chain
	chainPublicKey primitives.PublicKey
	queryStateKey  []byte

	// TCA KDFKey
	tCertOwnerKDFKey []byte
	tCertPool        tCertPool
}

// NewChaincodeDeployTransaction is used to deploy chaincode.
func (client *clientImpl) NewChaincodeDeployTransaction(chaincodeDeploymentSpec *obc.ChaincodeDeploymentSpec, uuid string, attributes ...string) (*obc.Transaction, error) {
	// Verify that the client is initialized
	if !client.isInitialized {
		return nil, utils.ErrNotInitialized
	}

	// Get next available (not yet used) transaction certificate
	tCerts, err := client.tCertPool.GetNextTCerts(1, attributes...)
	if err != nil {
		client.error("Failed to obtain a (not yet used) TCert for Chaincode Deploy[%s].", err.Error())
		return nil, err
	}

	if len(tCerts) != 1 {
		client.error("Failed to obtain a (not yet used) TCert.")
		return nil, errors.New("Failed to obtain a TCert for Chaincode Deploy Transaction using TCert. Expected exactly one returned TCert.")
	}

	// Create Transaction
	return client.newChaincodeDeployUsingTCert(chaincodeDeploymentSpec, uuid, attributes, tCerts[0].tCert, nil)
}

// GetNextTCerts Gets next available (not yet used) transaction certificate.
func (client *clientImpl) GetNextTCerts(nCerts int, attributes ...string) (tCerts []tCert, err error) {
	if nCerts < 1 {
		return nil, errors.New("Number of requested TCerts has to be positive!")
	}

	// Verify that the client is initialized
	if !client.isInitialized {
		return nil, utils.ErrNotInitialized
	}

	// Get next available (not yet used) transaction certificate
	tBlocks, err := client.tCertPool.GetNextTCerts(nCerts, attributes...)
	if err != nil {
		client.error("Failed getting [%d] (not yet used) Transaction Certificates (TCerts) [%s].", nCerts, err.Error())
		return nil, err
	}
	tCerts = make([]tCert, len(tBlocks))
	for i, eachBlock := range tBlocks {
		tCerts[i] = eachBlock.tCert
	}
	return tCerts, nil
}

// NewChaincodeInvokeTransaction is used to invoke chaincode's functions.
func (client *clientImpl) NewChaincodeExecute(chaincodeInvocation *obc.ChaincodeInvocationSpec, uuid string, attributes ...string) (*obc.Transaction, error) {
	// Verify that the client is initialized
	if !client.isInitialized {
		return nil, utils.ErrNotInitialized
	}

	// Get next available (not yet used) transaction certificate
	tBlocks, err := client.tCertPool.GetNextTCerts(1, attributes...)
	if err != nil {
		client.error("Failed to obtain a (not yet used) TCert [%s].", err.Error())
		return nil, err
	}

	if len(tBlocks) != 1 {
		client.error("Failed to obtain a (not yet used) TCert.")
		return nil, errors.New("Failed to obtain a TCert for Chaincode Execution. Expected exactly one returned TCert.")
	}

	// Create Transaction
	return client.newChaincodeExecuteUsingTCert(chaincodeInvocation, uuid, attributes, tBlocks[0].tCert, nil)
}

// NewChaincodeQuery is used to query chaincode's functions.
func (client *clientImpl) NewChaincodeQuery(chaincodeInvocation *obc.ChaincodeInvocationSpec, uuid string, attributes ...string) (*obc.Transaction, error) {
	// Verify that the client is initialized
	if !client.isInitialized {
		return nil, utils.ErrNotInitialized
	}

	// Get next available (not yet used) transaction certificate
	tBlocks, err := client.tCertPool.GetNextTCerts(1, attributes...)
	if err != nil {
		client.error("Failed to obtain a (not yet used) TCert [%s].", err.Error())
		return nil, err
	}

	if len(tBlocks) != 1 {
		client.error("Failed to obtain a (not yet used) TCert.")
		return nil, errors.New("Failed to obtain a TCert for Chaincode Invocation. Expected exactly one returned TCert.")
	}

	// Create Transaction
	return client.newChaincodeQueryUsingTCert(chaincodeInvocation, uuid, attributes, tBlocks[0].tCert, nil)
}

// GetEnrollmentCertHandler returns a CertificateHandler whose certificate is the enrollment certificate
func (client *clientImpl) GetEnrollmentCertificateHandler() (CertificateHandler, error) {
	// Verify that the client is initialized
	if !client.isInitialized {
		return nil, utils.ErrNotInitialized
	}

	// Return the handler
	handler := &eCertHandlerImpl{}
	err := handler.init(client)
	if err != nil {
		client.error("Failed getting handler [%s].", err.Error())
		return nil, err
	}

	return handler, nil
}

// GetTCertHandlerNext returns a CertificateHandler whose certificate is the next available TCert
func (client *clientImpl) GetTCertificateHandlerNext(attributes ...string) (CertificateHandler, error) {
	// Verify that the client is initialized
	if !client.isInitialized {
		return nil, utils.ErrNotInitialized
	}

	// Get next TCert
	tBlocks, err := client.tCertPool.GetNextTCerts(1, attributes...)
	if err != nil {
		client.error("Failed to obtain a (not yet used) TCert for creating a CertificateHandler [%s].", err.Error())
		return nil, err
	}

	if len(tBlocks) != 1 {
		client.error("Failed to obtain a TCert for creating a CertificateHandler.")
		return nil, errors.New("Failed to obtain a TCert for creating a CertificateHandler")
	}

	// Return the handler
	handler := &tCertHandlerImpl{}
	err = handler.init(client, tBlocks[0].tCert)
	if err != nil {
		client.error("Failed getting handler [%s].", err.Error())
		return nil, err
	}

	return handler, nil
}

// GetTCertHandlerFromDER returns a CertificateHandler whose certificate is the one passed
func (client *clientImpl) GetTCertificateHandlerFromDER(tCertDER []byte) (CertificateHandler, error) {
	// Verify that the client is initialized
	if !client.isInitialized {
		return nil, utils.ErrNotInitialized
	}

	// Validate the transaction certificate
	tCert, err := client.getTCertFromExternalDER(tCertDER)
	if err != nil {
		client.warning("Failed validating transaction certificate [%s].", err)

		return nil, err
	}

	// Return the handler
	handler := &tCertHandlerImpl{}
	err = handler.init(client, tCert)
	if err != nil {
		client.error("Failed getting handler [%s].", err.Error())
		return nil, err
	}

	return handler, nil
}

func (client *clientImpl) register(id string, pwd []byte, enrollID, enrollPWD string) (err error) {
	if client.isInitialized {
		client.error("Registering [%s]...done! Initialization already performed", id)

		return
	}

	// Register node
	if err = client.nodeImpl.register(NodeClient, id, pwd, enrollID, enrollPWD); err != nil {
		return
	}

	client.info("Register crypto engine...")
	err = client.registerCryptoEngine()
	if err != nil {
		log.Error("Failed registering crypto engine [%s]: [%s].", enrollID, err.Error())
		return
	}
	client.info("Register crypto engine...done.")

	return
}

func (client *clientImpl) init(id string, pwd []byte) error {
	if client.isInitialized {
		return utils.ErrAlreadyInitialized
	}

	// Register node
	if err := client.nodeImpl.init(NodeClient, id, pwd); err != nil {
		return err
	}

	// Initialize keystore
	client.debug("Init keystore...")
	err := client.initKeyStore()
	if err != nil {
		if err != utils.ErrKeyStoreAlreadyInitialized {
			client.error("Keystore already initialized.")
		} else {
			client.error("Failed initiliazing keystore [%s].", err.Error())

			return err
		}
	}
	client.debug("Init keystore...done.")

	// Init crypto engine
	err = client.initCryptoEngine()
	if err != nil {
		client.error("Failed initiliazing crypto engine [%s].", err.Error())
		return err
	}

	// initialized
	client.isInitialized = true

	return nil
}

func (client *clientImpl) close() (err error) {
	if client.tCertPool != nil {
		if err = client.tCertPool.Stop(); err != nil {
			client.debug("Failed closing TCertPool [%s]", err)
		}
	}

	if err = client.nodeImpl.close(); err != nil {
		client.debug("Failed closing node [%s]", err)
	}
	return
}
