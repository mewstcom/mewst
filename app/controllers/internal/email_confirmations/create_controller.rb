# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::CreateController < Internal::ApplicationController
  include Localizable
  include Latest::FormErrorable

  around_action :set_locale

  def call
    form = Internal::EmailConfirmationForm.new(
      email: params[:email],
      event: params[:event],
      locale: I18n.locale
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    result = CreateEmailConfirmationUseCase.new.call(
      email: form.email.not_nil!,
      event: form.event.not_nil!,
      locale: form.locale.not_nil!
    )

    resource = Internal::EmailConfirmationResource.new(email_confirmation: result.email_confirmation)
    render(
      json: Internal::EmailConfirmationSerializer.new(resource),
      status: :created
    )
  end
end
