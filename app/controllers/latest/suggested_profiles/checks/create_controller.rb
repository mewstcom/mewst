# typed: true
# frozen_string_literal: true

class Latest::SuggestedProfiles::Checks::CreateController < Latest::ApplicationController
  include Latest::Authenticatable
  include Latest::FormErrorable

  def call
    form = Latest::SuggestedFollowForm.new(
      viewer: current_viewer!,
      target_atname: params[:atname]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::SuggestedFollowFormErrorResource, errors: form.errors)
    end

    CheckSuggestedFollowUseCase.new.call(
      viewer: form.viewer.not_nil!,
      target_profile: form.target_profile.not_nil!
    )

    render(status: :no_content)
  end
end
