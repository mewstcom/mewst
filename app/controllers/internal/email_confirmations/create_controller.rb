# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::CreateController < Internal::ApplicationController
  def call
    form = Internal::EmailConfirmationForm.new(email: params[:email], locale: I18n.locale)

    if form.invalid?
      resources = Internal::FormErrorResource.build_from_errors(errors: form.errors)
      return render(
        json: Panko::Response.new(
          errors: Panko::ArraySerializer.new(resources, each_serializer: Internal::ResponseErrorSerializer)
        ),
        status: :unprocessable_entity
      )
    end

    result = CreateEmailConfirmationService.new(form:).call

    resource = Internal::EmailConfirmationResource.new(email_confirmation: result.email_confirmation)
    render(
      json: Panko::Response.new(
        email_confirmation: Internal::EmailConfirmationSerializer.new.serialize(resource)
      ),
      status: :created
    )
  end
end
