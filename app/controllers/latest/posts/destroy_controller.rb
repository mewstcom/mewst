# typed: true
# frozen_string_literal: true

class Latest::Posts::DestroyController < Latest::ApplicationController
  include Latest::Authenticatable
  include Latest::FormErrorable

  def call
    form = Latest::PostDeleteForm.new(
      viewer: current_viewer!,
      target_post_id: params[:post_id]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::PostFormErrorResource, errors: form.errors)
    end

    DeletePostUseCase.new.call(target_post: form.target_post.not_nil!)

    render(status: :no_content)
  end
end
