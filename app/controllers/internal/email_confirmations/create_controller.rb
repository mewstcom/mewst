# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::CreateController < Internal::ApplicationController
  def call
    form = Internal::EmailConfirmationForm.new(email: params[:email], locale: I18n.locale)

    if form.invalid?
      resources = Internal::FormErrorResource.build_from_errors(errors: form.errors)
      return render(
        json: Internal::ResponseErrorSerializer.new(resources),
        status: :unprocessable_entity
      )
    end

    input = CreateEmailConfirmationService::Input.from_internal_form(form:)
    result = CreateEmailConfirmationService.new.call(input:)

    resource = Internal::EmailConfirmationResource.new(email_confirmation: result.email_confirmation)
    render(
      json: Internal::EmailConfirmationSerializer.new(resource),
      status: :created
    )
  end
end
