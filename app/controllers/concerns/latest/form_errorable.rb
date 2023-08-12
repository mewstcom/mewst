# typed: strict
# frozen_string_literal: true

module Latest::FormErrorable
  extend T::Sig
  extend ActiveSupport::Concern

  sig { params(resource_class: T.class_of(Latest::FormErrorResource)).returns(T.untyped) }
  def response_form_errors(resource_class:, errors:)
    resource_errors = resource_class.build_from_errors(errors: form.errors)

    render(
      json: Panko::Response.new(
        errors: Panko::ArraySerializer.new(resource_errors, each_serializer: Latest::ResponseErrorSerializer)
      ),
      status: :unprocessable_entity
    )
  end
end
