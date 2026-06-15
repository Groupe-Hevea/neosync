import { yupResolver as hookformYupResolver } from '@hookform/resolvers/yup';
import { Resolver } from 'react-hook-form';
import * as Yup from 'yup';

/**
 * @hookform/resolvers v5 types a schema's input and output separately, so its
 * `yupResolver` returns `Resolver<Input, Context, Yup.InferType<Schema>>`. Across the
 * app, forms uniformly type their field values, `values` and `defaultValues` as
 * `Yup.InferType<typeof schema>`, which clashes with the distinct input type.
 *
 * This wrapper restores the single-type resolver semantics (input === output ===
 * `Yup.InferType`) the codebase is built around, keeping `useForm` call sites unchanged.
 */
export function yupResolver<TSchema extends Yup.ObjectSchema<Yup.AnyObject>>(
  schema: TSchema
): Resolver<Yup.InferType<TSchema>> {
  return hookformYupResolver(schema) as unknown as Resolver<
    Yup.InferType<TSchema>
  >;
}
