# typed: true
# frozen_string_literal: true

class Api::V1::Internal::EmailConfirmations::Challenges::CreateController < Api::V1::Internal::ApplicationController
  # include ControllerConcerns::InternalAuthenticatable
  # include ControllerConcerns::Localizable
  # include ControllerConcerns::V1::FormErrorable
  #
  # around_action :set_locale
  #
  # def call
  #   form = V1::EmailConfirmationChallengeForm.new(
  #     email_confirmation_id: params[:email_confirmation_id],
  #     confirmation_code: params[:confirmation_code]
  #   )
  #
  #   if form.invalid?
  #     return response_form_errors(resource_class: V1::FormErrorResource, errors: form.errors)
  #   end
  #
  #   result = ConfirmEmailUseCase.new.call(email_confirmation: form.email_confirmation.not_nil!)
  #
  #   resource = V1::EmailConfirmationResource.new(email_confirmation: result.email_confirmation)
  #   render(
  #     json: V1::EmailConfirmationSerializer.new(resource),
  #     status: :ok
  #   )
  # end
end
