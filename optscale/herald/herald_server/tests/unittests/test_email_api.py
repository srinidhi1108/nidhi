from herald_server.tests.unittests.test_herald_base import TestHeraldBase
from unittest.mock import patch


class TestEmailApi(TestHeraldBase):
    def setUp(self, *args):
        super().setUp()

    def test_email_send(self):
        email = ["test@hystax.com"]
        template_type = 'invite'
        template_params = {'message': "Email_message",
                           'customer_id': "b2ed3073-8c48-417f-9275-eb0d6cc5c7f1",
                           'object_name': "Object_name", 'object_type': "INFO",
                           'level': "INFO"}
        code, response = self.client_v2.email_send(
            email, "Email_subject", template_type, template_params)
        result = {
            'email': email,
            'subject': 'Email_subject',
            'template_type': template_type,
            'template_params': template_params
        }
        self.assertEqual(201, code)
        self.assertEquals(result, response)

    def test_email_hystax_registration_send(self):
        email = ["test@hystax.com"]
        template_type = 'new_employee'
        template_params = {'message': "Email_message",
                           'customer_id': "b2ed3073-8c48-417f-9275-eb0d6cc5c7f1",
                           'object_name': "Object_name", 'object_type': "INFO",
                           'level': "INFO", 'texts': {'user_email': 'test+1@gmail.com'}}
        publish_message_p_configured = patch(
            'herald_server.controllers.email.EmailController.publish_message'
        ).start()
        code, response = self.client_v2.email_send(
            email, "Email_subject", template_type, template_params)
        self.assertEqual(publish_message_p_configured.called, True)
        result = {
            'email': ["test@hystax.com"],
            'subject': 'Email_subject',
            'template_type': template_type,
            'template_params': template_params
        }
        self.assertEqual(201, code)
        self.assertEquals(result, response)

    def test_email_hystax_registration_not_publish(self):
        email = ["test@hystax.com"]
        template_type = 'new_employee'
        template_params = {'message': "Email_message",
                           'customer_id': "b2ed3073-8c48-417f-9275-eb0d6cc5c7f1",
                           'object_name': "Object_name", 'object_type': "INFO",
                           'level': "INFO", 'texts': {'user_email': 'test+1@hystax.com'}}
        publish_message_p_configured = patch(
            'herald_server.controllers.email.EmailController.publish_message'
        ).start()
        code, response = self.client_v2.email_send(
            email, "Email_subject", template_type, template_params)
        self.assertEqual(publish_message_p_configured.called, False)
        result = {
            'email': ["test@hystax.com"],
            'subject': 'Email_subject',
            'template_type': template_type,
            'template_params': template_params
        }
        self.assertEqual(201, code)
        self.assertEquals(result, response)

    def test_validation_email_payload(self):
        valid_email = 'test@hystax.com'
        code, response = self.client_v2.email_send(email=[valid_email],
                                                   subject=None)
        self.assertEqual(400, code)
        self.assertEquals('subject is not provided',
                          response['error']['reason'])
        code, response = self.client_v2.email_send(email=None, subject='subj')
        self.assertEqual(400, code)
        self.assertEquals('email is not provided', response['error']['reason'])
        invalid_email = ['invalid_email']
        code, response = self.client_v2.email_send(email=invalid_email,
                                                   subject='subj')
        self.assertEqual(400, code)
        self.assertEquals('invalid email', response['error']['reason'])
        code, response = self.client_v2.email_send(email=valid_email,
                                                   subject='subj')
        self.assertEqual(400, code)
        self.assertEquals('invalid email', response['error']['reason'])
        code, response = self.client_v2.email_send(
            email=[valid_email], subject='subj', template_type=None)
        self.assertEqual(400, code)
        self.assertEquals('template_type is not provided',
                          response['error']['reason'])
        code, response = self.client_v2.email_send(
            email=[valid_email], subject='subj',
            template_type='not_existing_type')
        self.assertEqual(400, code)
        self.assertEquals('invalid template_type', response['error']['reason'])
        invalid_template_params = 'template_params'
        code, response = self.client_v2.email_send(
            email=[valid_email], subject='subj',
            template_params=invalid_template_params)
        self.assertEqual(400, code)
        self.assertEquals('invalid template_params', response['error']['reason'])
