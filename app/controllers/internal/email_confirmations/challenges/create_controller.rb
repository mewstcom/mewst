# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::Challenges::CreateController < Internal::ApplicationController
  include Localizable
  include Latest::FormErrorable

  around_action :set_locale

  def call
    form = Internal::EmailConfirmationChallengeForm.new(
      email_confirmation_id: params[:email_confirmation_id],
      confirmation_code: params[:confirmation_code]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    result = ConfirmEmailUseCase.new.call(email_confirmation: form.email_confirmation.not_nil!)

    resource = Internal::EmailConfirmationResource.new(email_confirmation: result.email_confirmation)
    render(
      json: Internal::EmailConfirmationSerializer.new(resource),
      status: :ok
    )
  end
end
