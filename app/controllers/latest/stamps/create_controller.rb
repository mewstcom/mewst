# typed: true
# frozen_string_literal: true

class Latest::Stamps::CreateController < Latest::ApplicationController
  def call
    form = Forms::Stamp.new(
      profile: current_profile!,
      target_post_id: params[:post_id]
    )

    if form.invalid?
      resource_errors = Latest::Entities::StampFormError.build_from_errors(errors: form.errors)
      return render(
        json: Latest::Resources::ResponseError.new(resource_errors),
        status: :unprocessable_entity
      )
    end

    result = ActiveRecord::Base.transaction do
      Services::CreateStamp.new(form:).call
    end

    head :no_content
  end
end
