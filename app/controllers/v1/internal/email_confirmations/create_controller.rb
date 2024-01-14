# typed: true
# frozen_string_literal: true

class V1::Internal::EmailConfirmations::CreateController < V1::Internal::ApplicationController
  include InternalAuthenticatable
  include Localizable
  include V1::FormErrorable

  around_action :set_locale

  def call
    form = V1::EmailConfirmationForm.new(
      email: params[:email],
      event: params[:event],
      locale: I18n.locale
    )

    if form.invalid?
      return response_form_errors(resource_class: V1::FormErrorResource, errors: form.errors)
    end

    result = CreateEmailConfirmationUseCase.new.call(
      email: form.email.not_nil!,
      event: form.event.not_nil!,
      locale: form.locale.not_nil!
    )

    resource = V1::EmailConfirmationResource.new(email_confirmation: result.email_confirmation)
    render(
      json: V1::EmailConfirmationSerializer.new(resource),
      status: :created
    )
  end
end
