# typed: true
# frozen_string_literal: true

class Internal::Accounts::CreateController < Internal::ApplicationController
  def call
    form = Internal::AccountForm.new(
      atname: params[:atname],
      email: params[:email],
      locale: I18n.locale,
      password: params[:password]
    )

    if form.invalid?
      resources = Internal::FormErrorResource.build_from_errors(errors: form.errors)
      return render(
        json: Panko::Response.new(
          errors: Panko::ArraySerializer.new(resources, each_serializer: Internal::ResponseErrorSerializer)
        ),
        status: :unprocessable_entity
      )
    end

    input = CreateAccountService::Input.from_internal_form(form:)

    result = ActiveRecord::Base.transaction do
      result = CreateAccountService.new.call(input:)
      result.oauth_access_token.user!.track_sign_in
      result
    end

    resource = Internal::AccountResource.new(
      oauth_access_token: result.oauth_access_token,
      profile: result.profile,
      user: result.user
    )
    render(
      json: Panko::Response.new(
        account: Internal::AccountSerializer.new.serialize(resource)
      ),
      status: :created
    )
  end
end
