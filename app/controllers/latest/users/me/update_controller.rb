# typed: true
# frozen_string_literal: true

class Latest::Users::Me::UpdateController < Latest::ApplicationController
  include Latest::Authenticatable
  include Latest::FormErrorable

  def call
    form = Latest::UserForm.new(
      locale: params[:locale]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    result = UpdateUserUseCase.new.call(
      user: current_viewer!.user.not_nil!,
      locale: form.locale.not_nil!
    )

    user_resource = Latest::UserResource.new(user: result.user)
    render(
      json: {
        user: Latest::UserSerializer.new(user_resource).to_h
      },
      status: :ok
    )
  end
end
