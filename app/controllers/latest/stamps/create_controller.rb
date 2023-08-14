# typed: true
# frozen_string_literal: true

class Latest::Stamps::CreateController < Latest::ApplicationController
  include Latest::FormErrorable

  def call
    form = Latest::StampForm.new(
      profile: current_profile.not_nil!,
      target_post_id: params[:post_id]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::StampFormErrorResource, errors: form.errors)
    end

    ActiveRecord::Base.transaction do
      input = CreateStampService::Input.from_latest_form(form:)
      CreateStampService.new.call(input:)
    end

    head :no_content
  end
end
