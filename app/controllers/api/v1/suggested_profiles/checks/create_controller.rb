# typed: true
# frozen_string_literal: true

class V1::SuggestedProfiles::Checks::CreateController < V1::ApplicationController
  include ControllerConcerns::PublicAuthenticatable
  include ControllerConcerns::V1::FormErrorable

  def call
    form = V1::SuggestedFollowForm.new(
      viewer: current_viewer!,
      target_atname: params[:atname]
    )

    if form.invalid?
      return response_form_errors(resource_class: V1::SuggestedFollowFormErrorResource, errors: form.errors)
    end

    CheckSuggestedFollowUseCase.new.call(
      viewer: form.viewer.not_nil!,
      target_profile: form.target_profile.not_nil!
    )

    render(status: :no_content)
  end
end
